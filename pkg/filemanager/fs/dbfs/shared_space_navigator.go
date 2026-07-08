package dbfs

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

var (
	sharedSpaceAdminCapability  = &boolset.BooleanSet{}
	sharedSpaceEditorCapability = &boolset.BooleanSet{}
	sharedSpaceViewerCapability = &boolset.BooleanSet{}
)

func init() {
	// Admin: full access
	boolset.Sets(map[NavigatorCapability]bool{
		NavigatorCapabilityCreateFile:     true,
		NavigatorCapabilityRenameFile:     true,
		NavigatorCapabilityUploadFile:     true,
		NavigatorCapabilityDownloadFile:   true,
		NavigatorCapabilityUpdateMetadata: true,
		NavigatorCapabilityListChildren:   true,
		NavigatorCapabilityGenerateThumb:  true,
		NavigatorCapabilityDeleteFile:     true,
		NavigatorCapabilityLockFile:       true,
		NavigatorCapabilitySoftDelete:     true,
		NavigatorCapabilityRestore:        true,
		NavigatorCapabilityShare:          true,
		NavigatorCapabilityInfo:           true,
		NavigatorCapabilityVersionControl: true,
		NavigatorCapabilityEnterFolder:    true,
		NavigatorCapabilityModifyProps:    true,
	}, sharedSpaceAdminCapability)

	// Editor: write access without admin (no share, no restore)
	boolset.Sets(map[NavigatorCapability]bool{
		NavigatorCapabilityCreateFile:     true,
		NavigatorCapabilityRenameFile:     true,
		NavigatorCapabilityUploadFile:     true,
		NavigatorCapabilityDownloadFile:   true,
		NavigatorCapabilityUpdateMetadata: true,
		NavigatorCapabilityListChildren:   true,
		NavigatorCapabilityGenerateThumb:  true,
		NavigatorCapabilityDeleteFile:     true,
		NavigatorCapabilityLockFile:       true,
		NavigatorCapabilitySoftDelete:     true,
		NavigatorCapabilityInfo:           true,
		NavigatorCapabilityVersionControl: true,
		NavigatorCapabilityEnterFolder:    true,
		NavigatorCapabilityModifyProps:    true,
	}, sharedSpaceEditorCapability)

	// Viewer: read-only
	boolset.Sets(map[NavigatorCapability]bool{
		NavigatorCapabilityDownloadFile:   true,
		NavigatorCapabilityListChildren:   true,
		NavigatorCapabilityGenerateThumb:  true,
		NavigatorCapabilityInfo:           true,
		NavigatorCapabilityVersionControl: true,
		NavigatorCapabilityEnterFolder:    true,
	}, sharedSpaceViewerCapability)
}

// NewSharedSpaceNavigator creates a navigator for a shared space file system.
func NewSharedSpaceNavigator(u *ent.User, fileClient inventory.FileClient, spaceClient inventory.SpaceClient,
	l logging.Logger, config *setting.DBFS, hasher hashid.Encoder) Navigator {
	return &sharedSpaceNavigator{
		user:          u,
		l:             l,
		fileClient:    fileClient,
		spaceClient:   spaceClient,
		config:        config,
		hasher:        hasher,
		baseNavigator: newBaseNavigator(fileClient, defaultFilter, u, hasher, config),
	}
}

type sharedSpaceNavigator struct {
	l           logging.Logger
	user        *ent.User
	fileClient  inventory.FileClient
	spaceClient inventory.SpaceClient
	config      *setting.DBFS
	hasher      hashid.Encoder

	*baseNavigator
	root           *File
	space          *ent.SharedSpace
	role           types.SharedSpaceRole
	disableRecycle bool
	persist        func()
}

func (n *sharedSpaceNavigator) Recycle() {
	if n.persist != nil {
		n.persist()
		n.persist = nil
	}
	if n.root != nil && !n.disableRecycle {
		n.root.Recycle()
	}
}

func (n *sharedSpaceNavigator) PersistState(kv cache.Driver, key string) {
	n.disableRecycle = true
	n.persist = func() {
		kv.Set(key, n.root, ContextHintTTL)
	}
}

func (n *sharedSpaceNavigator) RestoreState(s State) error {
	n.disableRecycle = true
	if state, ok := s.(*File); ok {
		n.root = state
		return nil
	}

	return fmt.Errorf("invalid state type: %T", s)
}

// initRole looks up the shared space and user's role without initializing the file tree.
// This allows Capabilities() to return correct permissions before the full initSpace().
func (n *sharedSpaceNavigator) initRole(ctx context.Context, spaceHashID string) error {
	if n.role != "" {
		return nil
	}

	space, err := n.spaceClient.GetByHashID(ctx, spaceHashID)
	if err != nil {
		return fs.ErrPathNotExist.WithError(fmt.Errorf("共享空间 not found: %w", err))
	}

	// Check if user is a member
	role, err := n.spaceClient.GetMemberRole(ctx, space.ID, n.user.ID)
	if err != nil {
		return ErrPermissionDenied.WithError(fmt.Errorf("user is not a member of this 共享空间"))
	}

	n.space = space
	n.role = role
	return nil
}

func (n *sharedSpaceNavigator) initSpace(ctx context.Context, path *fs.URI) error {
	spaceHashID := path.ID("")
	if spaceHashID == "" {
		return fs.ErrPathNotExist.WithError(fmt.Errorf("invalid shared space hash id"))
	}

	// Initialize role first (idempotent if already called from getNavigator)
	if err := n.initRole(ctx, spaceHashID); err != nil {
		return err
	}

	// Get root file
	rootFile, err := n.spaceClient.Root(ctx, n.space)
	if err != nil {
		return fs.ErrPathNotExist.WithError(fmt.Errorf("space root folder not found: %w", err))
	}

	n.root = newFile(nil, rootFile)
	rootPath := path.Root()
	n.root.Path[pathIndexRoot], n.root.Path[pathIndexUser] = rootPath, rootPath
	if owner, err := rootFile.Edges.OwnerOrErr(); err == nil {
		n.root.OwnerModel = owner
	} else {
		n.root.OwnerModel = n.user
	}
	n.root.IsUserRoot = true
	n.root.CapabilitiesBs = n.Capabilities(false).Capability

	return nil
}

func (n *sharedSpaceNavigator) To(ctx context.Context, path *fs.URI) (*File, error) {
	if inventory.IsAnonymousUser(n.user) {
		return nil, ErrLoginRequired
	}

	if n.root == nil {
		if err := n.initSpace(ctx, path); err != nil {
			return nil, err
		}
	}

	current, lastAncestor := n.root, n.root
	elements := path.Elements()

	var err error
	for index, element := range elements {
		lastAncestor = current
		current, err = n.walkNext(ctx, current, element, index == len(elements)-1)
		if err != nil {
			return lastAncestor, fmt.Errorf("failed to walk into %q: %w", element, err)
		}
	}

	return current, nil
}

func (n *sharedSpaceNavigator) Children(ctx context.Context, parent *File, args *ListArgs) (*ListResult, error) {
	return n.baseNavigator.children(ctx, parent, args)
}

func (n *sharedSpaceNavigator) Capabilities(isSearching bool) *fs.NavigatorProps {
	var capabilities *boolset.BooleanSet
	switch n.role {
	case types.SharedSpaceRoleAdmin:
		capabilities = sharedSpaceAdminCapability
	case types.SharedSpaceRoleEditor:
		capabilities = sharedSpaceEditorCapability
	default:
		capabilities = sharedSpaceViewerCapability
	}

	res := &fs.NavigatorProps{
		Capability:            capabilities,
		OrderDirectionOptions: fullOrderDirectionOption,
		OrderByOptions:        fullOrderByOption,
		MaxPageSize:           n.config.MaxPageSize,
	}

	if isSearching {
		res.OrderByOptions = searchLimitedOrderByOption
	}

	return res
}

func (n *sharedSpaceNavigator) Walk(ctx context.Context, levelFiles []*File, limit, depth int, f WalkFunc) error {
	return n.baseNavigator.walk(ctx, levelFiles, limit, depth, f)
}

func (n *sharedSpaceNavigator) walkNext(ctx context.Context, root *File, next string, isLeaf bool) (*File, error) {
	return n.baseNavigator.walkNext(ctx, root, next, isLeaf)
}

func (n *sharedSpaceNavigator) FollowTx(ctx context.Context) (func(), error) {
	if _, ok := ctx.Value(inventory.TxCtx{}).(*inventory.Tx); !ok {
		return nil, fmt.Errorf("navigator: no inherited transaction found in context")
	}
	newFileClient, _, _, err := inventory.WithTx(ctx, n.fileClient)
	if err != nil {
		return nil, err
	}
	newSpaceClient, _, _, err := inventory.WithTx(ctx, n.spaceClient)
	if err != nil {
		return nil, err
	}

	oldFileClient, oldSpaceClient := n.fileClient, n.spaceClient
	revert := func() {
		n.fileClient = oldFileClient
		n.spaceClient = oldSpaceClient
		n.baseNavigator.fileClient = oldFileClient
	}

	n.fileClient = newFileClient
	n.spaceClient = newSpaceClient
	n.baseNavigator.fileClient = newFileClient
	return revert, nil
}

func (n *sharedSpaceNavigator) ExecuteHook(ctx context.Context, hookType fs.HookType, file *File) error {
	// Capability-based access control handles permissions.
	// Operations are gated by the navigator's capabilities which reflect the user's role.
	return nil
}

func (n *sharedSpaceNavigator) GetView(ctx context.Context, file *File) *types.ExplorerView {
	if view, ok := n.user.Settings.FsViewMap[string(constants.FileSystemSharedSpace)]; ok {
		return &view
	}
	return getDefaultView()
}

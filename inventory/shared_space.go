package inventory

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/file"
	"github.com/cloudreve/Cloudreve/v4/ent/sharedspace"
	"github.com/cloudreve/Cloudreve/v4/ent/sharedspacemember"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/samber/lo"
)

type (
	// LoadSpaceMembers is a context key for eager loading space members.
	LoadSpaceMembers struct{}
	// LoadSpaceFiles is a context key for eager loading space files.
	LoadSpaceFiles struct{}
)

type (
	// SpaceClient defines the interface for shared space operations.
	SpaceClient interface {
		TxOperator
		// Create creates a new shared space.
		Create(ctx context.Context, params *CreateSpaceParams) (*ent.SharedSpace, error)
		// GetByID returns the space with given id.
		GetByID(ctx context.Context, id int) (*ent.SharedSpace, error)
		// GetByHashID returns the space with given hash id.
		GetByHashID(ctx context.Context, hashID string) (*ent.SharedSpace, error)
		// ListSpaces returns all spaces the given user is a member of.
		ListSpaces(ctx context.Context, userID int, args *PaginationArgs) (*ListSpaceResult, error)
		// Update updates a shared space.
		Update(ctx context.Context, space *ent.SharedSpace) (*ent.SharedSpace, error)
		// Delete deletes a shared space.
		Delete(ctx context.Context, id int) error
		// AddMember adds a user or group to a shared space.
		AddMember(ctx context.Context, params *AddSpaceMemberParams) (*ent.SharedSpaceMember, error)
		// RemoveMember removes a member from a shared space.
		RemoveMember(ctx context.Context, memberID int) error
		// GetMemberByID returns a shared space member by id.
		GetMemberByID(ctx context.Context, memberID int) (*ent.SharedSpaceMember, error)
		// UpdateMemberRole updates a member's role.
		UpdateMemberRole(ctx context.Context, memberID int, role types.SharedSpaceRole) (*ent.SharedSpaceMember, error)
		// GetMemberRole returns the role of a user in a space.
		GetMemberRole(ctx context.Context, spaceID, userID int) (types.SharedSpaceRole, error)
		// ListMembers lists all members of a space.
		ListMembers(ctx context.Context, spaceID int, args *PaginationArgs) ([]*ent.SharedSpaceMember, *PaginationResults, error)
		// Root returns the root folder of a shared space.
		Root(ctx context.Context, space *ent.SharedSpace) (*ent.File, error)
	}

	// CreateSpaceParams contains parameters for creating a shared space.
	CreateSpaceParams struct {
		Name        string
		Description string
		OwnerID     int
	}

	// AddSpaceMemberParams contains parameters for adding a member to a space.
	AddSpaceMemberParams struct {
		SpaceID int
		UserID  int
		GroupID int
		Role    types.SharedSpaceRole
	}

	// ListSpaceResult is the result of listing shared spaces.
	ListSpaceResult struct {
		Spaces []*ent.SharedSpace
		*PaginationResults
	}

	// spaceClient implements SpaceClient.
	spaceClient struct {
		client *ent.Client
		hasher hashid.Encoder
	}
)

// NewSpaceClient creates a new SpaceClient.
func NewSpaceClient(client *ent.Client, hasher hashid.Encoder) SpaceClient {
	return &spaceClient{client: client, hasher: hasher}
}

func (c *spaceClient) SetClient(newClient *ent.Client) TxOperator {
	return &spaceClient{client: newClient, hasher: c.hasher}
}

func (c *spaceClient) GetClient() *ent.Client {
	return c.client
}

func (c *spaceClient) Create(ctx context.Context, params *CreateSpaceParams) (*ent.SharedSpace, error) {
	// Create the root file for the shared space
	rootFile, err := c.client.File.Create().
		SetOwnerID(params.OwnerID).
		SetType(int(types.FileTypeFolder)).
		SetName(RootFolderName).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create root folder for space: %w", err)
	}

	space, err := c.client.SharedSpace.Create().
		SetName(params.Name).
		SetDescription(params.Description).
		SetOwnerID(params.OwnerID).
		SetRootFileID(rootFile.ID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create shared space: %w", err)
	}

	// Add the owner as admin member
	_, err = c.client.SharedSpaceMember.Create().
		SetSharedSpaceID(space.ID).
		SetUserID(params.OwnerID).
		SetRole(sharedspacemember.RoleAdmin).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to add owner as space member: %w", err)
	}

	// Also add the owner's group as admin member so all group members can access
	owner, err := c.client.User.Get(ctx, params.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner: %w", err)
	}
	_, err = c.client.SharedSpaceMember.Create().
		SetSharedSpaceID(space.ID).
		SetGroupID(owner.GroupUsers).
		SetRole(sharedspacemember.RoleAdmin).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to add owner group as space member: %w", err)
	}

	return space, nil
}

func (c *spaceClient) GetByID(ctx context.Context, id int) (*ent.SharedSpace, error) {
	return c.client.SharedSpace.Get(ctx, id)
}

func (c *spaceClient) GetByHashID(ctx context.Context, hashID string) (*ent.SharedSpace, error) {
	id, err := c.hasher.Decode(hashID, hashid.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode space hash id: %w", err)
	}

	return c.GetByID(ctx, id)
}

func (c *spaceClient) ListSpaces(ctx context.Context, userID int, args *PaginationArgs) (*ListSpaceResult, error) {
	if args == nil {
		args = &PaginationArgs{}
	}
	pageSize := args.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}

	// Get user's group for group-level membership lookup
	user, err := c.client.User.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Find all space memberships for this user (both direct and group-level)
	memberships, err := c.client.SharedSpaceMember.Query().
		Where(
			sharedspacemember.Or(
				sharedspacemember.UserIDEQ(userID),
				sharedspacemember.GroupIDEQ(user.GroupUsers),
			),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query space memberships: %w", err)
	}

	spaceIDs := lo.Map(memberships, func(m *ent.SharedSpaceMember, _ int) int {
		return m.SharedSpaceID
	})

	if len(spaceIDs) == 0 {
		return &ListSpaceResult{
			Spaces: []*ent.SharedSpace{},
			PaginationResults: &PaginationResults{
				Page:       args.Page,
				PageSize:   pageSize,
				TotalItems: 0,
			},
		}, nil
	}

	query := c.client.SharedSpace.Query().
		Where(
			sharedspace.IDIn(spaceIDs...),
		)
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count spaces: %w", err)
	}

	spaces, err := query.
		Order(sharedspace.ByCreatedAt(sql.OrderDesc())).
		Limit(pageSize).
		Offset(args.Page * pageSize).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query spaces: %w", err)
	}

	return &ListSpaceResult{
		Spaces: spaces,
		PaginationResults: &PaginationResults{
			Page:       args.Page,
			PageSize:   pageSize,
			TotalItems: total,
		},
	}, nil
}

func (c *spaceClient) Update(ctx context.Context, space *ent.SharedSpace) (*ent.SharedSpace, error) {
	return c.client.SharedSpace.UpdateOne(space).
		SetName(space.Name).
		SetDescription(space.Description).
		Save(ctx)
}

func (c *spaceClient) Delete(ctx context.Context, id int) error {
	// Delete all memberships first
	_, err := c.client.SharedSpaceMember.Delete().
		Where(sharedspacemember.SharedSpaceIDEQ(id)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete space memberships: %w", err)
	}

	if err := c.client.SharedSpace.DeleteOneID(id).Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (c *spaceClient) AddMember(ctx context.Context, params *AddSpaceMemberParams) (*ent.SharedSpaceMember, error) {
	if params.UserID > 0 {
		existing, err := c.client.SharedSpaceMember.Query().
			Where(
				sharedspacemember.SharedSpaceIDEQ(params.SpaceID),
				sharedspacemember.UserIDEQ(params.UserID),
			).
			Only(ctx)
		if err == nil {
			return c.UpdateMemberRole(ctx, existing.ID, params.Role)
		}
		if !ent.IsNotFound(err) {
			return nil, err
		}
	}

	create := c.client.SharedSpaceMember.Create().
		SetSharedSpaceID(params.SpaceID).
		SetRole(sharedspacemember.Role(params.Role))
	if params.UserID > 0 {
		create.SetUserID(params.UserID)
	}
	if params.GroupID > 0 {
		create.SetGroupID(params.GroupID)
	}

	return create.Save(ctx)
}

func (c *spaceClient) RemoveMember(ctx context.Context, memberID int) error {
	return c.client.SharedSpaceMember.DeleteOneID(memberID).Exec(ctx)
}

func (c *spaceClient) GetMemberByID(ctx context.Context, memberID int) (*ent.SharedSpaceMember, error) {
	return c.client.SharedSpaceMember.Query().
		Where(sharedspacemember.IDEQ(memberID)).
		WithUser().
		WithGroup().
		Only(ctx)
}

func (c *spaceClient) UpdateMemberRole(ctx context.Context, memberID int, role types.SharedSpaceRole) (*ent.SharedSpaceMember, error) {
	return c.client.SharedSpaceMember.UpdateOneID(memberID).
		SetRole(sharedspacemember.Role(role)).
		Save(ctx)
}

func (c *spaceClient) GetMemberRole(ctx context.Context, spaceID, userID int) (types.SharedSpaceRole, error) {
	// First check direct user membership
	member, err := c.client.SharedSpaceMember.Query().
		Where(
			sharedspacemember.SharedSpaceIDEQ(spaceID),
			sharedspacemember.UserIDEQ(userID),
		).
		First(ctx)
	if err == nil {
		return types.SharedSpaceRole(member.Role), nil
	}
	if !ent.IsNotFound(err) {
		return "", fmt.Errorf("failed to get member role: %w", err)
	}

	// Check group membership — get user's group first
	user, err := c.client.User.Get(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user for group check: %w", err)
	}

	member, err = c.client.SharedSpaceMember.Query().
		Where(
			sharedspacemember.SharedSpaceIDEQ(spaceID),
			sharedspacemember.GroupIDEQ(user.GroupUsers),
		).
		First(ctx)
	if err != nil {
		return "", fmt.Errorf("user is not a member of this shared space: %w", err)
	}

	return types.SharedSpaceRole(member.Role), nil
}

func (c *spaceClient) ListMembers(ctx context.Context, spaceID int, args *PaginationArgs) ([]*ent.SharedSpaceMember, *PaginationResults, error) {
	if args == nil {
		args = &PaginationArgs{}
	}
	pageSize := args.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}

	query := c.client.SharedSpaceMember.Query().
		Where(sharedspacemember.SharedSpaceIDEQ(spaceID))
	total, err := query.Count(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count space members: %w", err)
	}

	members, err := query.
		WithUser().
		WithGroup().
		Order(sharedspacemember.ByCreatedAt(sql.OrderAsc())).
		Limit(pageSize).
		Offset(args.Page * pageSize).
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list space members: %w", err)
	}

	return members, &PaginationResults{Page: args.Page, PageSize: pageSize, TotalItems: total}, nil
}

func (c *spaceClient) Root(ctx context.Context, space *ent.SharedSpace) (*ent.File, error) {
	return c.client.File.Query().
		Where(file.IDEQ(space.RootFileID)).
		WithOwner(func(q *ent.UserQuery) {
			q.WithGroup()
		}).
		Only(ctx)
}

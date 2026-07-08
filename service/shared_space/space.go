package shared_space

import (
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/file"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	usersvc "github.com/cloudreve/Cloudreve/v4/service/user"
	"github.com/gin-gonic/gin"
)

var (
	ErrSpaceNotFound = serializer.NewError(serializer.CodeNotFound, "共享空间 not found", nil)
)

type (
	// CreateSharedSpaceService creates a shared space
	CreateSharedSpaceService struct {
		Name        string `json:"name" binding:"required,min=1,max=255"`
		Description string `json:"description" binding:"max=1000"`
	}

	// CreateSharedSpaceParamCtx for parameter passing
	CreateSharedSpaceParamCtx struct{}

	// SpaceIDService identifies a shared space.
	SpaceIDService struct {
		ID string `uri:"id" binding:"required"`
	}

	// SpaceIDParamCtx for parameter passing
	SpaceIDParamCtx struct{}

	// ListSharedSpaceService lists shared spaces for current user
	ListSharedSpaceService struct {
		Page     int `json:"page" form:"page"`
		PageSize int `json:"page_size" form:"page_size"`
	}

	// ListSharedSpaceParamCtx for parameter passing
	ListSharedSpaceParamCtx struct{}

	// AddMemberService adds a member to a shared space
	AddMemberService struct {
		UserID string `json:"user_id" binding:"required"`
		Role   string `json:"role" binding:"required,eq=admin|eq=editor|eq=viewer"`
	}

	// AddMemberParamCtx for parameter passing
	AddMemberParamCtx struct{}

	// UpdateMemberService updates a member role.
	UpdateMemberService struct {
		Role string `json:"role" binding:"required,eq=admin|eq=editor|eq=viewer"`
	}

	// UpdateMemberParamCtx for parameter passing
	UpdateMemberParamCtx struct{}

	// RemoveMemberService removes a member from a shared space
	RemoveMemberService struct {
	}

	// RemoveMemberParamCtx for parameter passing
	RemoveMemberParamCtx struct{}
)

func encodeSpace(dep dependency.Dep, space *ent.SharedSpace, role types.SharedSpaceRole) gin.H {
	res := gin.H{
		"id":          hashid.EncodeSpaceID(dep.HashIDEncoder(), space.ID),
		"name":        space.Name,
		"description": space.Description,
		"owner_id":    hashid.EncodeUserID(dep.HashIDEncoder(), space.OwnerID),
		"root_uri":    fmt.Sprintf("cloudreve://%s@shared_space", hashid.EncodeSpaceID(dep.HashIDEncoder(), space.ID)),
	}
	if role != "" {
		res["role"] = role
	}
	return res
}

func encodeMember(c *gin.Context, dep dependency.Dep, member *ent.SharedSpaceMember) gin.H {
	res := gin.H{
		"id":       hashid.EncodeSpaceMemberID(dep.HashIDEncoder(), member.ID),
		"role":     member.Role,
		"space_id": hashid.EncodeSpaceID(dep.HashIDEncoder(), member.SharedSpaceID),
	}
	if member.UserID != 0 {
		res["user_id"] = hashid.EncodeUserID(dep.HashIDEncoder(), member.UserID)
		if u, err := member.Edges.UserOrErr(); err == nil {
			res["user"] = usersvc.BuildUserRedacted(c, u, usersvc.RedactLevelUser, dep.HashIDEncoder())
		}
	}
	if member.GroupID != 0 {
		res["group_id"] = member.GroupID
		if g, err := member.Edges.GroupOrErr(); err == nil {
			res["group"] = gin.H{
				"id":   g.ID,
				"name": g.Name,
			}
		}
	}
	return res
}

func decodeSpaceID(c *gin.Context, dep dependency.Dep) (int, error) {
	spaceID, err := dep.HashIDEncoder().Decode(c.Param("id"), hashid.SpaceID)
	if err != nil {
		return 0, serializer.NewError(serializer.CodeNotFound, "Invalid space ID", err)
	}
	return spaceID, nil
}

func requireAdmin(c *gin.Context, dep dependency.Dep, spaceID int) (types.SharedSpaceRole, error) {
	user := inventory.UserFromContext(c)
	role, err := dep.SpaceClient().GetMemberRole(c, spaceID, user.ID)
	if err != nil {
		return "", serializer.NewError(serializer.CodeNoPermissionErr, "没有权限管理共享空间", err)
	}
	if role != types.SharedSpaceRoleAdmin {
		return "", serializer.NewError(serializer.CodeNoPermissionErr, "只有共享空间管理员可以管理成员", nil)
	}
	return role, nil
}

func requireMember(c *gin.Context, dep dependency.Dep, spaceID int) (types.SharedSpaceRole, error) {
	user := inventory.UserFromContext(c)
	role, err := dep.SpaceClient().GetMemberRole(c, spaceID, user.ID)
	if err != nil {
		return "", serializer.NewError(serializer.CodeNoPermissionErr, "没有权限访问共享空间", err)
	}
	return role, nil
}

func collectSpaceFiles(ctx *gin.Context, dep dependency.Dep, rootID int) ([]*ent.File, error) {
	files := make([]*ent.File, 0)
	current := []int{rootID}
	for len(current) > 0 {
		level, err := dep.DBClient().File.Query().
			Where(file.IDIn(current...)).
			WithEntities().
			All(ctx)
		if err != nil {
			return nil, err
		}
		files = append(files, level...)

		children, err := dep.DBClient().File.Query().
			Where(file.FileChildrenIn(current...)).
			All(ctx)
		if err != nil {
			return nil, err
		}
		current = make([]int, 0, len(children))
		for _, child := range children {
			current = append(current, child.ID)
		}
	}

	return files, nil
}

// Create creates a new shared space
func (s *CreateSharedSpaceService) Create(c *gin.Context) (*serializer.Response, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)

	spaceClient, tx, ctx, err := inventory.WithTx(c, dep.SpaceClient())
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}

	space, err := spaceClient.Create(ctx, &inventory.CreateSpaceParams{
		Name:        s.Name,
		Description: s.Description,
		OwnerID:     user.ID,
	})
	if err != nil {
		_ = inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeDBError, "创建共享空间失败", err)
	}
	if err := inventory.Commit(tx); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "提交共享空间失败", err)
	}

	return &serializer.Response{
		Data: encodeSpace(dep, space, types.SharedSpaceRoleAdmin),
	}, nil
}

// List lists shared spaces for the current user
func (s *ListSharedSpaceService) List(c *gin.Context) (*serializer.Response, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)

	result, err := dep.SpaceClient().ListSpaces(c, user.ID, &inventory.PaginationArgs{
		Page:     s.Page,
		PageSize: s.PageSize,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list shared spaces", err)
	}

	spaces := make([]gin.H, 0, len(result.Spaces))
	for _, space := range result.Spaces {
		role, _ := dep.SpaceClient().GetMemberRole(c, space.ID, user.ID)
		spaces = append(spaces, encodeSpace(dep, space, role))
	}

	return &serializer.Response{
		Data: gin.H{
			"spaces":     spaces,
			"pagination": result.PaginationResults,
		},
	}, nil
}

// Update updates a shared space.
func (s *CreateSharedSpaceService) Update(c *gin.Context) (*serializer.Response, error) {
	dep := dependency.FromContext(c)
	spaceID, err := decodeSpaceID(c, dep)
	if err != nil {
		return nil, err
	}
	if _, err := requireAdmin(c, dep, spaceID); err != nil {
		return nil, err
	}

	space, err := dep.SpaceClient().GetByID(c, spaceID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "共享空间 not found", err)
	}
	space.Name = s.Name
	space.Description = s.Description
	space, err = dep.SpaceClient().Update(c, space)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update shared space", err)
	}

	return &serializer.Response{Data: encodeSpace(dep, space, types.SharedSpaceRoleAdmin)}, nil
}

// Delete deletes a shared space.
func (s *SpaceIDService) Delete(c *gin.Context) (*serializer.Response, error) {
	dep := dependency.FromContext(c)
	spaceID, err := dep.HashIDEncoder().Decode(s.ID, hashid.SpaceID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Invalid space ID", err)
	}
	if _, err := requireAdmin(c, dep, spaceID); err != nil {
		return nil, err
	}
	space, err := dep.SpaceClient().GetByID(c, spaceID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "共享空间 not found", err)
	}

	spaceClient, tx, ctx, err := inventory.WithTx(c, dep.SpaceClient())
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}
	fileClient, _ := inventory.InheritTx(ctx, dep.FileClient())
	userClient, _ := inventory.InheritTx(ctx, dep.UserClient())

	files, err := collectSpaceFiles(c, dep, space.RootFileID)
	if err != nil {
		_ = inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to collect shared space files", err)
	}
	if err := spaceClient.Delete(ctx, spaceID); err != nil {
		_ = inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to delete shared space", err)
	}
	if len(files) > 0 {
		_, storageDiff, err := fileClient.Delete(ctx, files, &types.EntityProps{})
		if err != nil {
			_ = inventory.Rollback(tx)
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to delete shared space files", err)
		}
		if err := userClient.ApplyStorageDiff(ctx, storageDiff); err != nil {
			_ = inventory.Rollback(tx)
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to apply storage diff", err)
		}
	}
	if err := inventory.Commit(tx); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "提交共享空间删除失败", err)
	}

	return &serializer.Response{}, nil
}

// Members lists shared space members.
func (s *ListSharedSpaceService) Members(c *gin.Context) (*serializer.Response, error) {
	dep := dependency.FromContext(c)
	spaceID, err := decodeSpaceID(c, dep)
	if err != nil {
		return nil, err
	}
	if _, err := requireMember(c, dep, spaceID); err != nil {
		return nil, err
	}

	members, pagination, err := dep.SpaceClient().ListMembers(c, spaceID, &inventory.PaginationArgs{
		Page:     s.Page,
		PageSize: s.PageSize,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list shared space members", err)
	}

	return &serializer.Response{
		Data: gin.H{
			"members":    loMapMembers(c, dep, members),
			"pagination": pagination,
		},
	}, nil
}

func loMapMembers(c *gin.Context, dep dependency.Dep, members []*ent.SharedSpaceMember) []gin.H {
	res := make([]gin.H, 0, len(members))
	for _, member := range members {
		res = append(res, encodeMember(c, dep, member))
	}
	return res
}

// Add adds a member to a shared space
func (s *AddMemberService) Add(c *gin.Context) (*serializer.Response, error) {
	dep := dependency.FromContext(c)
	spaceID, err := decodeSpaceID(c, dep)
	if err != nil {
		return nil, err
	}
	if _, err := requireAdmin(c, dep, spaceID); err != nil {
		return nil, err
	}

	userID, err := dep.HashIDEncoder().Decode(s.UserID, hashid.UserID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Invalid user ID", err)
	}

	member, err := dep.SpaceClient().AddMember(c, &inventory.AddSpaceMemberParams{
		SpaceID: spaceID,
		UserID:  userID,
		Role:    types.SharedSpaceRole(s.Role),
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to add member", err)
	}
	member, _ = dep.SpaceClient().GetMemberByID(c, member.ID)

	return &serializer.Response{
		Data: encodeMember(c, dep, member),
	}, nil
}

// Update updates a member role.
func (s *UpdateMemberService) Update(c *gin.Context) (*serializer.Response, error) {
	dep := dependency.FromContext(c)
	memberID, err := dep.HashIDEncoder().Decode(c.Param("memberId"), hashid.SpaceMemberID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Invalid member ID", err)
	}
	member, err := dep.SpaceClient().GetMemberByID(c, memberID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Member not found", err)
	}
	if _, err := requireAdmin(c, dep, member.SharedSpaceID); err != nil {
		return nil, err
	}
	space, err := dep.SpaceClient().GetByID(c, member.SharedSpaceID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "共享空间 not found", err)
	}
	if member.UserID == space.OwnerID && types.SharedSpaceRole(s.Role) != types.SharedSpaceRoleAdmin {
		return nil, serializer.NewError(serializer.CodeNoPermissionErr, "Space owner must keep admin role", nil)
	}

	member, err = dep.SpaceClient().UpdateMemberRole(c, memberID, types.SharedSpaceRole(s.Role))
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update member role", err)
	}
	member, _ = dep.SpaceClient().GetMemberByID(c, member.ID)

	return &serializer.Response{Data: encodeMember(c, dep, member)}, nil
}

// Remove removes a member from a shared space
func (s *RemoveMemberService) Remove(c *gin.Context) (*serializer.Response, error) {
	dep := dependency.FromContext(c)
	memberIDStr := c.Param("memberId")
	memberID, err := dep.HashIDEncoder().Decode(memberIDStr, hashid.SpaceMemberID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Invalid member ID", err)
	}
	member, err := dep.SpaceClient().GetMemberByID(c, memberID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Member not found", err)
	}
	if _, err := requireAdmin(c, dep, member.SharedSpaceID); err != nil {
		return nil, err
	}
	space, err := dep.SpaceClient().GetByID(c, member.SharedSpaceID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "共享空间 not found", err)
	}
	if member.UserID == space.OwnerID {
		return nil, serializer.NewError(serializer.CodeNoPermissionErr, "Space owner cannot be removed", nil)
	}

	if err := dep.SpaceClient().RemoveMember(c, memberID); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to remove member", err)
	}

	return &serializer.Response{}, nil
}

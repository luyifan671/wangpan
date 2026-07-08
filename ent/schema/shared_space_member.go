package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// SharedSpaceMember holds the schema definition for the SharedSpaceMember entity.
// Represents a user or group membership in a shared space with a specific role.
type SharedSpaceMember struct {
	ent.Schema
}

// Fields of the SharedSpaceMember.
func (SharedSpaceMember) Fields() []ent.Field {
	return []ent.Field{
		field.Int("shared_space_id"),
		field.Int("user_id").
			Optional(),
		field.Int("group_id").
			Optional(),
		field.Enum("role").
			Values("admin", "editor", "viewer").
			Default("viewer"),
	}
}

// Edges of the SharedSpaceMember.
func (SharedSpaceMember) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("shared_space", SharedSpace.Type).
			Ref("members").
			Field("shared_space_id").
			Unique().
			Required(),
		edge.From("user", User.Type).
			Ref("space_memberships").
			Field("user_id").
			Unique(),
		edge.From("group", Group.Type).
			Ref("space_memberships").
			Field("group_id").
			Unique(),
	}
}

func (SharedSpaceMember) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// SharedSpace holds the schema definition for the SharedSpace entity.
// SharedSpace represents a team/shared drive where multiple users can
// collaboratively store and manage files in a shared directory tree.
type SharedSpace struct {
	ent.Schema
}

// Fields of the SharedSpace.
func (SharedSpace) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(255),
		field.String("description").
			Optional().
			MaxLen(1000),
		field.Int("owner_id"),
		field.Int("root_file_id").
			Optional(),
	}
}

// Edges of the SharedSpace.
func (SharedSpace) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner", User.Type).
			Ref("owned_spaces").
			Field("owner_id").
			Unique().
			Required(),
		edge.To("members", SharedSpaceMember.Type),
		edge.To("root_file", File.Type).
			Field("root_file_id").
			Unique(),
	}
}

func (SharedSpace) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}

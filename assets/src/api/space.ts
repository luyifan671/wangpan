import { PaginationResults } from "./explorer.ts";
import { User } from "./user.ts";

export type SharedSpaceRole = "admin" | "editor" | "viewer";

export interface SharedSpace {
  id: string;
  name: string;
  description?: string;
  owner_id: string;
  role?: SharedSpaceRole;
  root_uri: string;
}

export interface SharedSpaceMember {
  id: string;
  space_id: string;
  user_id?: string;
  group_id?: number;
  role: SharedSpaceRole;
  user?: User;
  group?: {
    id: number;
    name: string;
  };
  users?: SharedSpaceMemberUser[];
  via_group_name?: string;
}

export interface SharedSpaceMemberUser {
  user_id: string;
  user: User;
}

export interface ListSharedSpaceService {
  page?: number;
  page_size?: number;
}

export interface ListSharedSpaceResponse {
  spaces: SharedSpace[];
  pagination: PaginationResults;
}

export interface CreateSharedSpaceService {
  name: string;
  description?: string;
}

export interface AddSharedSpaceMemberService {
  user_id: string;
  role: SharedSpaceRole;
}

export interface UpdateSharedSpaceMemberService {
  role: SharedSpaceRole;
}

export interface ListSharedSpaceMembersResponse {
  members: SharedSpaceMember[];
  pagination: PaginationResults;
}

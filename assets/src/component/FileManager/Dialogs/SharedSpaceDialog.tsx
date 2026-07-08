import { LoadingButton } from "@mui/lab";
import {
  Avatar,
  Box,
  DialogContent,
  Divider,
  IconButton,
  List,
  ListItem,
  ListItemAvatar,
  ListItemText,
  MenuItem,
  Select,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from "@mui/material";
import { DeleteOutline } from "@mui/icons-material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  getSharedSpaceMembers,
  sendAddSharedSpaceMember,
  sendCreateSharedSpace,
  sendRemoveSharedSpaceMember,
  sendUpdateSharedSpace,
  sendUpdateSharedSpaceMember,
} from "../../../api/api.ts";
import { SharedSpace, SharedSpaceMember, SharedSpaceRole } from "../../../api/space.ts";
import { User } from "../../../api/user.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import UserAvatar from "../../Common/User/UserAvatar.tsx";
import UserSearchInput from "../../Admin/File/UserSearchInput.tsx";

export interface SharedSpaceDialogProps {
  open: boolean;
  space?: SharedSpace;
  onClose: () => void;
  onChanged?: (space?: SharedSpace) => void;
}

const roleOptions: SharedSpaceRole[] = ["admin", "editor", "viewer"];

const roleLabel = (role: SharedSpaceRole) => {
  switch (role) {
    case "admin":
      return "Admin";
    case "editor":
      return "Editor";
    default:
      return "Viewer";
  }
};

const SharedSpaceDialog = ({ open, space, onClose, onChanged }: SharedSpaceDialogProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [members, setMembers] = useState<SharedSpaceMember[]>([]);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [selectedRole, setSelectedRole] = useState<SharedSpaceRole>("editor");
  const [loading, setLoading] = useState(false);
  const [memberLoading, setMemberLoading] = useState(false);

  const isManage = !!space;
  const canManageMembers = space?.role === "admin";

  const loadMembers = useCallback(async () => {
    if (!space) {
      setMembers([]);
      return;
    }

    setMemberLoading(true);
    try {
      const res = await dispatch(getSharedSpaceMembers(space.id, { page_size: 200 }));
      setMembers(res.members);
    } finally {
      setMemberLoading(false);
    }
  }, [dispatch, space]);

  useEffect(() => {
    if (!open) {
      return;
    }

    setName(space?.name ?? "");
    setDescription(space?.description ?? "");
    setSelectedUser(null);
    setSelectedRole("editor");
    loadMembers();
  }, [open, space, loadMembers]);

  const title = useMemo(() => {
    if (isManage) {
      return t("application:sharedSpace.manage", { defaultValue: "管理共享空间" });
    }
    return t("application:sharedSpace.create", { defaultValue: "创建共享空间" });
  }, [isManage, t]);

  const onAccept = useCallback(async () => {
    setLoading(true);
    try {
      const req = { name, description };
      const updated = space ? await dispatch(sendUpdateSharedSpace(space.id, req)) : await dispatch(sendCreateSharedSpace(req));
      onChanged?.(updated);
      if (!space) {
        onClose();
      }
    } finally {
      setLoading(false);
    }
  }, [description, dispatch, name, onChanged, onClose, space]);

  const onAddMember = useCallback(async () => {
    if (!space || !selectedUser) {
      return;
    }

    setMemberLoading(true);
    try {
      await dispatch(sendAddSharedSpaceMember(space.id, { user_id: selectedUser.id, role: selectedRole }));
      setSelectedUser(null);
      await loadMembers();
      onChanged?.(space);
    } finally {
      setMemberLoading(false);
    }
  }, [dispatch, loadMembers, onChanged, selectedRole, selectedUser, space]);

  const onChangeRole = useCallback(
    async (member: SharedSpaceMember, role: SharedSpaceRole) => {
      if (!space) {
        return;
      }

      setMemberLoading(true);
      try {
        await dispatch(sendUpdateSharedSpaceMember(space.id, member.id, { role }));
        await loadMembers();
      } finally {
        setMemberLoading(false);
      }
    },
    [dispatch, loadMembers, space],
  );

  const onRemoveMember = useCallback(
    async (member: SharedSpaceMember) => {
      if (!space) {
        return;
      }

      setMemberLoading(true);
      try {
        await dispatch(sendRemoveSharedSpaceMember(space.id, member.id));
        await loadMembers();
      } finally {
        setMemberLoading(false);
      }
    },
    [dispatch, loadMembers, space],
  );

  return (
    <DraggableDialog
      title={title}
      loading={loading}
      disabled={name.trim().length === 0}
      dialogProps={{ open, onClose }}
      showActions
      showCancel
      okText={isManage ? t("common:save", { defaultValue: "保存" }) : t("common:create", { defaultValue: "Create" })}
      onAccept={onAccept}
    >
      <DialogContent sx={{ pt: 1, minWidth: { xs: "unset", sm: 460 } }}>
        <Stack spacing={2}>
          <TextField
            autoFocus
            fullWidth
            size="small"
            label={t("application:sharedSpace.name", { defaultValue: "Name" })}
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <TextField
            fullWidth
            multiline
            minRows={2}
            size="small"
            label={t("application:sharedSpace.description", { defaultValue: "Description" })}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
          {isManage && (
            <>
              <Divider />
              <Typography variant="subtitle2">
                {t("application:sharedSpace.members", { defaultValue: "Members" })}
              </Typography>
              {canManageMembers && (
                <Stack direction={{ xs: "column", sm: "row" }} spacing={1} alignItems="stretch">
                  <Box sx={{ flex: 1 }}>
                    <UserSearchInput
                      label={t("application:sharedSpace.searchUser", { defaultValue: "Search user" })}
                      onUserSelected={setSelectedUser}
                    />
                  </Box>
                  <Select
                    size="small"
                    value={selectedRole}
                    onChange={(e) => setSelectedRole(e.target.value as SharedSpaceRole)}
                  >
                    {roleOptions.map((role) => (
                      <MenuItem key={role} value={role}>
                        {roleLabel(role)}
                      </MenuItem>
                    ))}
                  </Select>
                  <LoadingButton
                    variant="outlined"
                    loading={memberLoading}
                    disabled={!selectedUser}
                    onClick={onAddMember}
                  >
                    {t("common:add", { defaultValue: "Add" })}
                  </LoadingButton>
                </Stack>
              )}
              <List dense disablePadding>
                {members.map((member) => (
                  <ListItem
                    key={member.id}
                    disableGutters
                    secondaryAction={
                      canManageMembers && (
                        <Stack direction="row" spacing={1} alignItems="center">
                          <Select
                            size="small"
                            value={member.role}
                            disabled={memberLoading}
                            onChange={(e) => onChangeRole(member, e.target.value as SharedSpaceRole)}
                          >
                            {roleOptions.map((role) => (
                              <MenuItem key={role} value={role}>
                                {roleLabel(role)}
                              </MenuItem>
                            ))}
                          </Select>
                          <Tooltip title={t("common:delete")}>
                            <IconButton disabled={memberLoading} onClick={() => onRemoveMember(member)}>
                              <DeleteOutline fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        </Stack>
                      )
                    }
                  >
                    <ListItemAvatar>
                      {member.user ? <UserAvatar user={member.user} /> : <Avatar />}
                    </ListItemAvatar>
                    <ListItemText
                      primary={member.user?.nickname ?? member.user_id}
                      secondary={member.user?.email ?? roleLabel(member.role)}
                    />
                  </ListItem>
                ))}
              </List>
            </>
          )}
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};

export default SharedSpaceDialog;

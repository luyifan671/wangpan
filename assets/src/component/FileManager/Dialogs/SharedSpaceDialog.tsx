import {
  Avatar,
  DialogContent,
  Divider,
  List,
  ListItem,
  ListItemAvatar,
  ListItemText,
  Stack,
  TextField,
  Typography,
} from "@mui/material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  getSharedSpaceMembers,
  sendCreateSharedSpace,
  sendUpdateSharedSpace,
} from "../../../api/api.ts";
import { SharedSpace, SharedSpaceMember, SharedSpaceRole } from "../../../api/space.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import SessionManager from "../../../session";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import UserAvatar from "../../Common/User/UserAvatar.tsx";

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
  const [loading, setLoading] = useState(false);
  const [memberLoading, setMemberLoading] = useState(false);

  const currentUserId = SessionManager.currentLoginOrNull()?.user?.id;
  const isManage = !!space;
  const isOwner = space?.owner_id === currentUserId;

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

  return (
    <DraggableDialog
      title={title}
      loading={loading}
      disabled={name.trim().length === 0}
      dialogProps={{ open, onClose }}
      showActions
      showCancel
      okText={isManage ? t("common:save", { defaultValue: "保存" }) : t("common:create", { defaultValue: "创建" })}
      onAccept={onAccept}
    >
      <DialogContent sx={{ pt: 1, minWidth: { xs: "unset", sm: 460 } }}>
        <Stack spacing={2}>
          <TextField
            autoFocus
            fullWidth
            size="small"
            label={t("application:sharedSpace.name", { defaultValue: "名称" })}
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <TextField
            fullWidth
            multiline
            minRows={2}
            size="small"
            label={t("application:sharedSpace.description", { defaultValue: "描述" })}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
          {isManage && (
            <>
              <Divider />
              <Typography variant="subtitle2">
                {t("application:sharedSpace.members", { defaultValue: "成员" })}
              </Typography>
              <List dense disablePadding>
                {members.map((member) => {
                  const isViaGroup = !!member.via_group_name;

                  return (
                    <ListItem
                      key={`${member.id}-${member.user_id ?? member.id}`}
                      disableGutters
                    >
                      <ListItemAvatar>
                        {member.user ? <UserAvatar user={member.user} /> : <Avatar />}
                      </ListItemAvatar>
                      <ListItemText
                        primary={member.user?.nickname ?? member.user_id}
                        secondary={
                          isViaGroup
                            ? t("application:sharedSpace.inheritedFromGroup", { defaultValue: "继承自用户组" }) + ` ${member.via_group_name}`
                            : (member.user?.email ?? roleLabel(member.role))
                        }
                      />
                    </ListItem>
                  );
                })}
              </List>
            </>
          )}
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};

export default SharedSpaceDialog;

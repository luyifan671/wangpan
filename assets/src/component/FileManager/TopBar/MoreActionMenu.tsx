import { ListItemIcon, ListItemText, MenuProps, Typography, useMediaQuery, useTheme } from "@mui/material";
import { GroupsOutlined } from "@mui/icons-material";
import { usePopupState } from "material-ui-popup-state/hooks";
import { useCallback, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { getSharedSpaces } from "../../../api/api.ts";
import { SharedSpace } from "../../../api/space.ts";
import { clearSelected } from "../../../redux/fileManagerSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { createShareShortcut, isMacbook } from "../../../redux/thunks/file.ts";
import { inverseSelection, pinCurrentView, refreshFileList, selectAll } from "../../../redux/thunks/filemanager.ts";
import SessionManager from "../../../session";
import { Filesystem } from "../../../util/uri.ts";
import { KeyIndicator } from "../../Frame/NavBar/SearchBar.tsx";
import ArrowSync from "../../Icons/ArrowSync.tsx";
import Border from "../../Icons/Border.tsx";
import BorderAll from "../../Icons/BorderAll.tsx";
import BorderInside from "../../Icons/BorderInside.tsx";
import FolderLink from "../../Icons/FolderLink.tsx";
import PinOutlined from "../../Icons/PinOutlined.tsx";
import { DenseDivider, SquareMenu, SquareMenuItem } from "../ContextMenu/ContextMenu.tsx";
import SharedSpaceDialog from "../Dialogs/SharedSpaceDialog.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";

const MoreActionMenu = ({ onClose, ...rest }: MenuProps) => {
  const { t } = useTranslation();
  const fmIndex = useContext(FmIndexContext);
  const fs = useAppSelector((state) => state.fileManager[fmIndex].current_fs);
  const elements = useAppSelector((state) => state.fileManager[fmIndex].path_elements);
  const dispatch = useAppDispatch();
  const [spaceDialogOpen, setSpaceDialogOpen] = useState(false);
  const [selectedSpace, setSelectedSpace] = useState<SharedSpace | undefined>();
  const isLogin = !!SessionManager.currentLoginOrNull();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));

  const mountPopupState = usePopupState({
    variant: "popover",
    popupId: "mount",
  });

  const onPinClicked = useCallback(() => {
    dispatch(pinCurrentView(fmIndex));
    onClose && onClose({}, "escapeKeyDown");
  }, [dispatch, onClose, fmIndex]);

  const onCreateShortcutClicked = useCallback(() => {
    dispatch(createShareShortcut(fmIndex));
    onClose && onClose({}, "escapeKeyDown");
  }, [dispatch, onClose, fmIndex]);

  const onSelectAllClicked = useCallback(() => {
    onClose && onClose({}, "escapeKeyDown");
    dispatch(selectAll(fmIndex));
  }, [dispatch, onClose, fmIndex]);

  const onSelectNoneClicked = useCallback(() => {
    onClose && onClose({}, "escapeKeyDown");
    dispatch(clearSelected({ index: fmIndex, value: undefined }));
  }, [dispatch, onClose, fmIndex]);

  const onInverseSelectionClicked = useCallback(() => {
    onClose && onClose({}, "escapeKeyDown");
    dispatch(inverseSelection(fmIndex));
  }, [dispatch, onClose, fmIndex]);

  const onRefreshClicked = useCallback(() => {
    onClose && onClose({}, "escapeKeyDown");
    dispatch(refreshFileList(fmIndex));
  }, [dispatch, onClose, fmIndex]);

  const onManageSharedSpaceClicked = useCallback(async () => {
    onClose && onClose({}, "escapeKeyDown");
    const spaceID = elements?.[0];
    if (!spaceID) {
      return;
    }

    const res = await dispatch(getSharedSpaces({ page_size: 200 }));
    const space = res.spaces.find((s) => s.id === spaceID);
    if (space) {
      setSelectedSpace(space);
      setSpaceDialogOpen(true);
    }
  }, [dispatch, elements, onClose]);

  return (
    <>
      <SquareMenu
        MenuListProps={{
          dense: true,
        }}
        anchorOrigin={{
          vertical: "bottom",
          horizontal: "right",
        }}
        transformOrigin={{
          vertical: "top",
          horizontal: "right",
        }}
        onClose={onClose}
        {...rest}
      >
        {isMobile && (
          <>
            <SquareMenuItem onClick={onRefreshClicked}>
              <ListItemIcon>
                <ArrowSync fontSize="small" />
              </ListItemIcon>
              <ListItemText>{t("application:fileManager.refresh")}</ListItemText>
            </SquareMenuItem>
          </>
        )}
        {isLogin && (
          <SquareMenuItem onClick={onPinClicked}>
            <ListItemIcon>
              <PinOutlined fontSize="small" />
            </ListItemIcon>
            <ListItemText>{t("application:fileManager.pin")}</ListItemText>
          </SquareMenuItem>
        )}
        {isLogin && fs == Filesystem.share && (
          <SquareMenuItem onClick={onCreateShortcutClicked}>
            <ListItemIcon>
              <FolderLink fontSize="small" />
            </ListItemIcon>
            <ListItemText>{t("application:fileManager.saveShortcut")}</ListItemText>
          </SquareMenuItem>
        )}
        {isLogin && fs == Filesystem.shared_space && (
          <SquareMenuItem onClick={onManageSharedSpaceClicked}>
            <ListItemIcon>
              <GroupsOutlined fontSize="small" />
            </ListItemIcon>
            <ListItemText>{t("application:sharedSpace.manage", { defaultValue: "管理共享空间" })}</ListItemText>
          </SquareMenuItem>
        )}
        {isLogin && <DenseDivider />}
        <SquareMenuItem onClick={onSelectAllClicked}>
          <ListItemIcon>
            <BorderAll fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.selectAll")}</ListItemText>
          <Typography variant="body2" color="text.secondary">
            <KeyIndicator>{isMacbook ? "⌘" : "Ctrl"}</KeyIndicator>+<KeyIndicator>A</KeyIndicator>
          </Typography>
        </SquareMenuItem>
        <SquareMenuItem onClick={onSelectNoneClicked}>
          <ListItemIcon>
            <Border fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.selectNone")}</ListItemText>
        </SquareMenuItem>
        <SquareMenuItem onClick={onInverseSelectionClicked}>
          <ListItemIcon>
            <BorderInside fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.invertSelection")}</ListItemText>
        </SquareMenuItem>
      </SquareMenu>
      <SharedSpaceDialog
        open={spaceDialogOpen}
        space={selectedSpace}
        onClose={() => setSpaceDialogOpen(false)}
        onChanged={(space) => setSelectedSpace(space)}
      />
    </>
  );
};

export default MoreActionMenu;

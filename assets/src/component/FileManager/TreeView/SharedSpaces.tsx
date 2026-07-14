import { GroupsOutlined } from "@mui/icons-material";
import { Collapse } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { TransitionGroup } from "react-transition-group";
import { getSharedSpaces, sendCreateSharedSpace } from "../../../api/api.ts";
import { SharedSpace } from "../../../api/space.ts";
import SessionManager from "../../../session";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { navigateToPath } from "../../../redux/thunks/filemanager.ts";
import CrUri, { Filesystem } from "../../../util/uri.ts";
import SideNavItem from "../../Frame/NavBar/SideNavItem.tsx";
import SharedSpaceDialog from "../Dialogs/SharedSpaceDialog.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";
import { SquareMenu, SquareMenuItem } from "../ContextMenu/ContextMenu.tsx";
import { ListItemIcon, ListItemText } from "@mui/material";
import SettingsOutlined from "../../Icons/SettingsOutlined.tsx";
import TreeFiles from "./TreeFiles.tsx";

const sharedSpacesChangedEvent = "cloudreve:shared-spaces-changed";

export const SharedSpacesNavItem = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const fmIndex = useContext(FmIndexContext);
  const currentFs = useAppSelector((s) => s.fileManager[fmIndex].current_fs);

  const handleClick = useCallback(async () => {
    const res = await dispatch(getSharedSpaces({ page_size: 1 }));
    let space: SharedSpace | undefined = res.spaces[0];
    if (!space) {
      const groupName = SessionManager.currentUserGroup()?.name || "";
      space = await dispatch(sendCreateSharedSpace({ name: groupName }));
      window.dispatchEvent(new Event(sharedSpacesChangedEvent));
    }
    dispatch(navigateToPath(fmIndex, space.root_uri));
  }, [dispatch, fmIndex]);

  return (
    <SideNavItem
      level={0}
      icon={<GroupsOutlined fontSize="small" color="action" />}
      label={t("application:sharedSpace.title", { defaultValue: "共享空间" })}
      active={currentFs === Filesystem.shared_space}
      onClick={handleClick}
    />
  );
};

const SharedSpaces = () => {
  const dispatch = useAppDispatch();
  const fmIndex = useContext(FmIndexContext);
  const [spaces, setSpaces] = useState<SharedSpace[]>([]);
  const currentFs = useAppSelector((s) => s.fileManager[fmIndex].current_fs);
  const path = useAppSelector((s) => s.fileManager[fmIndex].path);
  const elements = useAppSelector((s) => s.fileManager[fmIndex].path_elements);
  const [manageOpen, setManageOpen] = useState(false);
  const [manageSpace, setManageSpace] = useState<SharedSpace | undefined>();
  const [contextMenu, setContextMenu] = useState<{
    open: boolean;
    x: number;
    y: number;
    space?: SharedSpace;
  }>({ open: false, x: 0, y: 0 });
  const { t } = useTranslation();

  const loadSpaces = useCallback(async () => {
    const res = await dispatch(getSharedSpaces({ page_size: 200 }));
    setSpaces(res.spaces);
  }, [dispatch]);

  useEffect(() => {
    loadSpaces();
  }, [loadSpaces]);

  useEffect(() => {
    window.addEventListener(sharedSpacesChangedEvent, loadSpaces);
    return () => window.removeEventListener(sharedSpacesChangedEvent, loadSpaces);
  }, [loadSpaces]);

  const activeSpaceID = useMemo(() => {
    if (currentFs !== Filesystem.shared_space || !path) {
      return undefined;
    }
    return new CrUri(path).id();
  }, [currentFs, path]);

  const onContextMenu = useCallback(
    (space: SharedSpace) => (e: React.MouseEvent<HTMLElement>) => {
      e.preventDefault();
      e.stopPropagation();
      setContextMenu({ open: true, x: e.clientX, y: e.clientY, space });
    },
    [],
  );

  const onContextMenuClose = useCallback(() => {
    setContextMenu((prev) => ({ ...prev, open: false }));
  }, []);

  const onManageClick = useCallback(() => {
    setManageSpace(contextMenu.space);
    setManageOpen(true);
    onContextMenuClose();
  }, [contextMenu.space, onContextMenuClose]);

  const onManageChanged = useCallback(
    (space?: SharedSpace) => {
      setManageOpen(false);
      if (space) {
        loadSpaces();
      }
    },
    [loadSpaces],
  );

  return (
    <>
      <SharedSpaceDialog
        open={manageOpen}
        space={manageSpace}
        onClose={() => setManageOpen(false)}
        onChanged={onManageChanged}
      />
      <SquareMenu
        open={contextMenu.open}
        onClose={onContextMenuClose}
        anchorReference="anchorPosition"
        anchorPosition={contextMenu.open ? { top: contextMenu.y, left: contextMenu.x } : undefined}
        MenuListProps={{ dense: true }}
      >
        <SquareMenuItem onClick={onManageClick}>
          <ListItemIcon>
            <SettingsOutlined fontSize="small" />
          </ListItemIcon>
          <ListItemText>
            {t("application:sharedSpace.manage", { defaultValue: "管理共享空间" })}
          </ListItemText>
        </SquareMenuItem>
      </SquareMenu>
      <TransitionGroup>
        {spaces.map((space) => {
          const spaceElements =
            activeSpaceID && activeSpaceID === new CrUri(space.root_uri).id() ? elements : undefined;
          return (
            <Collapse key={space.id}>
              <TreeFiles
                canDrop={space.role === "admin" || space.role === "editor"}
                level={1}
                path={space.root_uri}
                labelOverwrite={space.name}
                elements={spaceElements}
                onContextMenuOverride={onContextMenu(space)}
              />
            </Collapse>
          );
        })}
      </TransitionGroup>
    </>
  );
};

export default SharedSpaces;

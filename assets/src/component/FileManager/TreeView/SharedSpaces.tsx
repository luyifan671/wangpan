import { GroupsOutlined } from "@mui/icons-material";
import { Collapse } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { TransitionGroup } from "react-transition-group";
import { getSharedSpaces } from "../../../api/api.ts";
import { SharedSpace } from "../../../api/space.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { navigateToPath } from "../../../redux/thunks/filemanager.ts";
import CrUri, { Filesystem } from "../../../util/uri.ts";
import SideNavItem from "../../Frame/NavBar/SideNavItem.tsx";
import SharedSpaceDialog from "../Dialogs/SharedSpaceDialog.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";
import TreeFiles from "./TreeFiles.tsx";

const sharedSpacesChangedEvent = "cloudreve:shared-spaces-changed";

export const SharedSpacesNavItem = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const fmIndex = useContext(FmIndexContext);
  const [dialogOpen, setDialogOpen] = useState(false);
  const currentFs = useAppSelector((s) => s.fileManager[fmIndex].current_fs);

  const onChanged = useCallback(
    async (space?: SharedSpace) => {
      window.dispatchEvent(new Event(sharedSpacesChangedEvent));
      setDialogOpen(false);
      if (space) {
        dispatch(navigateToPath(fmIndex, space.root_uri));
      }
    },
    [dispatch, fmIndex],
  );

  return (
    <>
      <SideNavItem
        level={0}
        icon={<GroupsOutlined fontSize="small" color="action" />}
        label={t("application:sharedSpace.title", { defaultValue: "共享空间" })}
        active={currentFs === Filesystem.shared_space}
        onClick={() => setDialogOpen(true)}
      />
      <SharedSpaceDialog open={dialogOpen} onClose={() => setDialogOpen(false)} onChanged={onChanged} />
    </>
  );
};

const SharedSpaces = () => {
  const dispatch = useAppDispatch();
  const fmIndex = useContext(FmIndexContext);
  const [spaces, setSpaces] = useState<SharedSpace[]>([]);
  const currentFs = useAppSelector((s) => s.fileManager[fmIndex].current_fs);
  const path = useAppSelector((s) => s.fileManager[fmIndex].path);
  const elements = useAppSelector((s) => s.fileManager[fmIndex].path_elements);

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

  return (
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
            />
          </Collapse>
        );
      })}
    </TransitionGroup>
  );
};

export default SharedSpaces;

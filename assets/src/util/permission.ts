import { FileResponse } from "../api/explorer.ts";
import CrUri, { Filesystem } from "./uri.ts";

// canCopyMoveTo checks if the files can be copied or moved to the destination.
export function canCopyMoveTo(files: FileResponse[], dst: string, isCopy: boolean): boolean {
  const dstUri = new CrUri(dst);
  const srcUri = new CrUri(files[0].path);
  const sameSharedSpace =
    srcUri.fs() == Filesystem.shared_space && dstUri.fs() == Filesystem.shared_space && srcUri.id() == dstUri.id();
  if (isCopy) {
    return (
      (srcUri.fs() == dstUri.fs() && (srcUri.fs() == Filesystem.my || sameSharedSpace)) ||
      (srcUri.fs() == Filesystem.my && dstUri.fs() == Filesystem.shared_space)
    );
  } else {
    switch (srcUri.fs()) {
      case Filesystem.my:
        return dstUri.fs() == Filesystem.my || dstUri.fs() == Filesystem.trash;
      case Filesystem.trash:
        return dstUri.fs() == Filesystem.my;
      case Filesystem.shared_space:
        return sameSharedSpace;
    }
  }

  return false;
}

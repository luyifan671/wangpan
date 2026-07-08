import { Dialog } from "@mui/material";

export interface ProDialogProps {
  open: boolean;
  onClose: () => void;
}

// Pro features are hidden - dialog renders nothing
const ProDialog = (_props: ProDialogProps) => {
  return <Dialog open={false} onClose={_props.onClose} />;
};

export default ProDialog;

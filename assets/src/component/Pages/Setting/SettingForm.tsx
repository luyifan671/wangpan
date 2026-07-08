import { Chip, Grid2, Typography, styled } from "@mui/material";
import { useEffect } from "react";

export const ProChip = styled(Chip)(({ theme }) => ({
  marginLeft: 8,
  height: "20px",
  fontSize: "12px",
  background: `linear-gradient(45deg, ${theme.palette.primary.main} 30%, ${theme.palette.primary.light} 90%)`,
  color: theme.palette.primary.contrastText,
  display: "none",
}));

export interface SettingFormProps {
  title?: React.ReactNode;
  children: React.ReactNode;
  lgWidth?: number;
  secondary?: React.ReactNode;
  spacing?: number;
  anchorId?: string;
  noContainer?: boolean;
  pro?: boolean;
}

const SettingForm = ({
  title,
  children,
  lgWidth = 8,
  secondary,
  spacing,
  noContainer,
  anchorId,
  pro,
}: SettingFormProps) => {
  useEffect(() => {
    if (anchorId && window.location.hash === `#${anchorId}`) {
      const anchor = document.getElementById(`anchor-${anchorId}`);
      if (anchor) {
        anchor.scrollIntoView({ behavior: "smooth" });
        // clear hash, not query
        window.history.replaceState({}, "", window.location.pathname + window.location.search);
      }
    }
  }, [anchorId]);

  // Hide pro-only settings
  if (pro) {
    return null;
  }

  const inner = (
    <>
      <Grid2
        sx={{
          boxShadow: anchorId && window.location.hash === `#${anchorId}` ? "0 0 0 3px rgb(255 193 7 / 53%)" : "none",
        }}
        size={{
          md: lgWidth,
          xs: 12,
        }}
      >
        {title && (
          <Typography
            fontWeight={600}
            sx={{ mb: 0.5, display: "flex", alignItems: "center" }}
            variant={"body2"}
            id={anchorId ? `anchor-${anchorId}` : undefined}
          >
            {title}
          </Typography>
        )}
        <div>{children}</div>
      </Grid2>
      {secondary && secondary}
    </>
  );
  if (noContainer) {
    return inner;
  }
  return (
    <Grid2 container spacing={spacing ?? 0}>
      {inner}
    </Grid2>
  );
};

export default SettingForm;

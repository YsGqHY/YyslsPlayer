import { TextField } from '@mui/material';

interface NumberFieldProps {
  value: number;
  min?: number;
  max?: number;
  step?: number;
  onChange: (value: number) => void;
}

export const NumberField = ({ value, min, max, step = 1, onChange }: NumberFieldProps) => (
  <TextField
    type="number"
    value={value}
    onChange={(event) => onChange(Number(event.target.value))}
    fullWidth
    size="small"
    slotProps={{ htmlInput: { min, max, step } }}
  />
);

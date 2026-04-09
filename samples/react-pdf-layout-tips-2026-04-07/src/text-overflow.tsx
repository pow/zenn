import { View, Text, StyleSheet } from "@react-pdf/renderer";

const styles = StyleSheet.create({
  container: {
    padding: 8,
    border: "1pt solid #e2e8f0",
    borderRadius: 4,
    marginBottom: 8,
  },
  heading: {
    fontSize: 12,
    fontWeight: "bold",
    marginBottom: 4,
  },
  clampedText: {
    fontSize: 10,
    lineHeight: 1.5,
  },
  fixedWidthCell: {
    width: 120,
    padding: 4,
  },
});

type ClampedTextProps = {
  text: string;
  maxLines?: number;
};

export const ClampedText = ({ text, maxLines = 2 }: ClampedTextProps) => (
  <Text style={styles.clampedText} maxLines={maxLines} ellipsis="…">
    {text}
  </Text>
);

type TruncatedCellProps = {
  label: string;
  value: string;
  width?: number;
};

export const TruncatedCell = ({
  label,
  value,
  width = 120,
}: TruncatedCellProps) => (
  <View style={[styles.fixedWidthCell, { width }]}>
    <Text style={styles.heading}>{label}</Text>
    <Text style={styles.clampedText} maxLines={1} ellipsis="…">
      {value}
    </Text>
  </View>
);

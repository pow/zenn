import { View, Text, StyleSheet } from "@react-pdf/renderer";

const styles = StyleSheet.create({
  container: {
    flexDirection: "row",
    height: 24,
    borderRadius: 4,
    overflow: "hidden",
  },
  segment: {
    justifyContent: "center",
    alignItems: "center",
  },
  label: {
    fontSize: 8,
    color: "#ffffff",
  },
});

type Segment = {
  label: string;
  value: number;
  color: string;
};

type SegmentBarProps = {
  segments: Segment[];
};

export const SegmentBar = ({ segments }: SegmentBarProps) => {
  const total = segments.reduce((sum, s) => sum + s.value, 0);

  return (
    <View style={styles.container}>
      {segments.map((segment, i) => {
        const ratio = total > 0 ? segment.value / total : 0;
        return (
          <View
            key={i}
            style={[
              styles.segment,
              {
                flexBasis: `${(ratio * 100).toFixed(1)}%`,
                flexGrow: 0,
                flexShrink: 0,
                backgroundColor: segment.color,
              },
            ]}
          >
            <Text style={styles.label}>{segment.label}</Text>
          </View>
        );
      })}
    </View>
  );
};

export const normalizeSegments = (segments: Segment[]): Segment[] => {
  const total = segments.reduce((sum, s) => sum + s.value, 0);
  if (total === 0) return segments;
  return segments.map((s) => ({
    ...s,
    value: Math.round((s.value / total) * 100),
  }));
};

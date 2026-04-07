import { View, Text, StyleSheet } from "@react-pdf/renderer";

const styles = StyleSheet.create({
  card: {
    border: "1pt solid #e2e8f0",
    borderRadius: 4,
    padding: 12,
    marginBottom: 8,
  },
  title: {
    fontSize: 14,
    fontWeight: "bold",
    marginBottom: 6,
  },
  body: {
    fontSize: 10,
    lineHeight: 1.6,
  },
});

type CardSectionProps = {
  title: string;
  body: string;
};

export const CardSection = ({ title, body }: CardSectionProps) => (
  <View wrap={false} style={styles.card}>
    <Text style={styles.title}>{title}</Text>
    <Text style={styles.body}>{body}</Text>
  </View>
);

type CardListProps = {
  items: CardSectionProps[];
};

export const CardList = ({ items }: CardListProps) => (
  <View>
    {items.map((item, i) => (
      <CardSection key={i} title={item.title} body={item.body} />
    ))}
  </View>
);

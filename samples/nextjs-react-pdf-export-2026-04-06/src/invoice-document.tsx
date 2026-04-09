import { Document, Page, Text, View, StyleSheet } from "@react-pdf/renderer";

const styles = StyleSheet.create({
  page: { padding: 40, fontSize: 12 },
  title: { fontSize: 20, marginBottom: 20 },
  row: {
    flexDirection: "row",
    justifyContent: "space-between",
    marginBottom: 8,
  },
  total: {
    flexDirection: "row",
    justifyContent: "space-between",
    marginTop: 16,
    borderTopWidth: 1,
    borderTopColor: "#333",
    paddingTop: 8,
    fontWeight: "bold",
  },
});

export type Item = { name: string; amount: number };

export type InvoiceProps = {
  invoiceNo: string;
  items: Item[];
};

export function InvoiceDocument({ invoiceNo, items }: InvoiceProps) {
  const total = items.reduce((sum, item) => sum + item.amount, 0);
  return (
    <Document>
      <Page size="A4" style={styles.page}>
        <Text style={styles.title}>請求書 #{invoiceNo}</Text>
        {items.map((item, i) => (
          <View key={i} style={styles.row}>
            <Text>{item.name}</Text>
            <Text>¥{item.amount.toLocaleString()}</Text>
          </View>
        ))}
        <View style={styles.total}>
          <Text>合計</Text>
          <Text>¥{total.toLocaleString()}</Text>
        </View>
      </Page>
    </Document>
  );
}

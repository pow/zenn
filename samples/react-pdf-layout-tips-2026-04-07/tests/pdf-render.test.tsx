import { describe, it, expect } from "vitest";
import { renderToBuffer } from "@react-pdf/renderer";
import { Document, Page, View } from "@react-pdf/renderer";
import { CardSection, CardList } from "../src/card-section";
import { SegmentBar, normalizeSegments } from "../src/segment-bar";
import { ClampedText, TruncatedCell } from "../src/text-overflow";

const wrap = (children: React.ReactNode) => (
  <Document>
    <Page size="A4" style={{ padding: 24 }}>
      <View>{children}</View>
    </Page>
  </Document>
);

describe("CardSection - ページ跨ぎ防止", () => {
  it("CardSection が PDF としてレンダリングできる", async () => {
    const buffer = await renderToBuffer(
      wrap(
        <CardSection
          title="テスト項目"
          body="wrap={false} によりこのカードはページをまたがず、次のページに移動します。"
        />
      )
    );
    expect(buffer).toBeInstanceOf(Buffer);
    expect(buffer.length).toBeGreaterThan(0);
  });

  it("CardList が複数カードを含む PDF をレンダリングできる", async () => {
    const items = Array.from({ length: 5 }, (_, i) => ({
      title: `項目 ${i + 1}`,
      body: `説明文 ${i + 1}: このカードは wrap={false} でページ跨ぎを防止しています。`,
    }));
    const buffer = await renderToBuffer(wrap(<CardList items={items} />));
    expect(buffer).toBeInstanceOf(Buffer);
    expect(buffer.length).toBeGreaterThan(0);
  });
});

describe("SegmentBar - セグメント幅の正規化", () => {
  it("SegmentBar が PDF としてレンダリングできる", async () => {
    const segments = [
      { label: "A", value: 30, color: "#3b82f6" },
      { label: "B", value: 50, color: "#10b981" },
      { label: "C", value: 20, color: "#f59e0b" },
    ];
    const buffer = await renderToBuffer(
      wrap(<SegmentBar segments={segments} />)
    );
    expect(buffer).toBeInstanceOf(Buffer);
    expect(buffer.length).toBeGreaterThan(0);
  });

  it("normalizeSegments が合計100%になるよう正規化する", () => {
    const segments = [
      { label: "A", value: 3, color: "#3b82f6" },
      { label: "B", value: 5, color: "#10b981" },
      { label: "C", value: 2, color: "#f59e0b" },
    ];
    const normalized = normalizeSegments(segments);
    const total = normalized.reduce((sum, s) => sum + s.value, 0);
    expect(total).toBe(100);
    expect(normalized[0].value).toBe(30);
    expect(normalized[1].value).toBe(50);
    expect(normalized[2].value).toBe(20);
  });

  it("normalizeSegments が value=0 のケースを処理できる", () => {
    const segments = [
      { label: "A", value: 0, color: "#3b82f6" },
      { label: "B", value: 0, color: "#10b981" },
    ];
    const normalized = normalizeSegments(segments);
    expect(normalized[0].value).toBe(0);
    expect(normalized[1].value).toBe(0);
  });
});

describe("ClampedText / TruncatedCell - テキスト溢れ制御", () => {
  it("ClampedText が PDF としてレンダリングできる", async () => {
    const longText =
      "これは非常に長いテキストです。maxLines と ellipsis を使うことで、指定した行数を超えた部分は省略記号で表示されます。PDF のレイアウトを崩さずにテキストを制御できます。";
    const buffer = await renderToBuffer(
      wrap(<ClampedText text={longText} maxLines={2} />)
    );
    expect(buffer).toBeInstanceOf(Buffer);
    expect(buffer.length).toBeGreaterThan(0);
  });

  it("TruncatedCell が PDF としてレンダリングできる", async () => {
    const buffer = await renderToBuffer(
      wrap(
        <TruncatedCell
          label="名前"
          value="非常に長い名前がここに入りますが、セルの幅に収まるよう省略されます"
          width={150}
        />
      )
    );
    expect(buffer).toBeInstanceOf(Buffer);
    expect(buffer.length).toBeGreaterThan(0);
  });
});

import { describe, it, expect } from "vitest";
import { renderInvoicePdf } from "../src/render-pdf.js";
import type { InvoiceProps } from "../src/invoice-document.js";

describe("renderInvoicePdf", () => {
  it("generates a valid PDF buffer", async () => {
    const buffer = await renderInvoicePdf({
      invoiceNo: "2026-001",
      items: [
        { name: "Web サイト制作", amount: 300000 },
        { name: "保守・運用（3ヶ月）", amount: 90000 },
      ],
    });

    expect(buffer).toBeInstanceOf(Buffer);
    expect(buffer.length).toBeGreaterThan(0);
    // PDF files start with %PDF magic bytes
    expect(buffer.subarray(0, 5).toString()).toBe("%PDF-");
  });

  it("generates PDF with single item", async () => {
    const buffer = await renderInvoicePdf({
      invoiceNo: "2026-002",
      items: [{ name: "コンサルティング", amount: 50000 }],
    });

    expect(buffer).toBeInstanceOf(Buffer);
    expect(buffer.subarray(0, 5).toString()).toBe("%PDF-");
  });
});

describe("event loop blocking", () => {
  const props: InvoiceProps = {
    invoiceNo: "blocking-test",
    items: Array.from({ length: 30 }, (_, i) => ({
      name: `Item ${i + 1}`,
      amount: 1000 * (i + 1),
    })),
  };

  it("Promise.all does not speed up concurrent renders because layout computation blocks the main thread", async () => {
    // Warm up to ensure fonts and modules are cached
    await renderInvoicePdf(props);

    // Sequential: render two PDFs one after another
    const seqStart = performance.now();
    await renderInvoicePdf(props);
    await renderInvoicePdf(props);
    const seqTime = performance.now() - seqStart;

    // "Concurrent": render two PDFs with Promise.all
    // If renderToBuffer were truly async, this should be ~2x faster.
    // But because Yoga layout computation is synchronous, the event loop
    // is blocked and the two renders execute sequentially anyway.
    const concStart = performance.now();
    await Promise.all([renderInvoicePdf(props), renderInvoicePdf(props)]);
    const concTime = performance.now() - concStart;

    // Concurrent should NOT be significantly faster than sequential,
    // proving that the main thread is blocked during rendering.
    // If it were truly async, concTime would be ~0.5x seqTime.
    expect(concTime).toBeGreaterThan(seqTime * 0.5);
  });
});

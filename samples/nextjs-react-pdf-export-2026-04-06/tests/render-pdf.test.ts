import { describe, it, expect } from "vitest";
import { renderInvoicePdf } from "../src/render-pdf.js";

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

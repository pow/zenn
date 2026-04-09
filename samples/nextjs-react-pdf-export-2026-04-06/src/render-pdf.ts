import { renderToBuffer } from "@react-pdf/renderer";
import { createElement } from "react";
import { InvoiceDocument, type InvoiceProps } from "./invoice-document.js";

export async function renderInvoicePdf(props: InvoiceProps): Promise<Buffer> {
  const buffer = await renderToBuffer(
    createElement(InvoiceDocument, props) as any
  );
  return Buffer.from(buffer);
}

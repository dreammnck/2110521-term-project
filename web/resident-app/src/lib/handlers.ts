import { createHandlers } from "@condo/shared/server";

export const handlers = createHandlers({ cookieName: "condo_resident_token", role: "resident" });

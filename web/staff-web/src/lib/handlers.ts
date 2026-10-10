import { createHandlers } from "@condo/shared/server";

export const handlers = createHandlers({ cookieName: "condo_staff_token", role: "staff" });

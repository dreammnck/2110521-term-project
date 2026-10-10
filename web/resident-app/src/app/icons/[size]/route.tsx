import { ImageResponse } from "next/og";

// Generates the app icon at any size (?maskable=1 for full-bleed Android adaptive icons).
export async function GET(req: Request, ctx: { params: Promise<{ size: string }> }) {
  const { size } = await ctx.params;
  const n = Math.min(Math.max(parseInt(size, 10) || 192, 48), 1024);
  const maskable = new URL(req.url).searchParams.has("maskable");
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          background: "#9b1c31",
          color: "white",
          fontSize: n * (maskable ? 0.42 : 0.55),
          fontWeight: 700,
          borderRadius: maskable ? 0 : n * 0.22,
        }}
      >
        C
      </div>
    ),
    { width: n, height: n, headers: { "Cache-Control": "public, max-age=86400" } },
  );
}

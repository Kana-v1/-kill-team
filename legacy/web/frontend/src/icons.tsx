// Original geometric operative marks — no third-party artwork. An operative's
// `photo` (a data URI) replaces the glyph when present. The inner markup is
// carried verbatim from the original single-file tracker.

import type { Operative } from "./types";

const ICONS: Record<string, string> = {
  laurel:
    '<path d="M6 19c-3.2-4-2.4-9.6 1.8-12.6M18 19c3.2-4 2.4-9.6-1.8-12.6"/><path d="M12 6.5l1.7 3.4 3.8.5-2.8 2.7.7 3.7L12 15.1 8.6 16.8l.7-3.7-2.8-2.7 3.8-.5z"/>',
  chevron2: '<path d="M4 10.5l8-5 8 5M4 17l8-5 8 5"/>',
  rifle: '<path d="M3 13h12l3-3h3M6.5 13v4M12 10v3"/>',
  launcher: '<path d="M3 11.5h11l3-3h3M6.5 11.5v3"/><circle cx="11" cy="17" r="3"/>',
  chainsword:
    '<path d="M4.5 20l9.5-9.5 3 3L7.5 23z" transform="translate(0,-3)"/><path d="M14 7.5l2.5-2.5 3.5 3.5-2.5 2.5"/><path d="M6.5 14.5l1.5 1.5M9.5 11.5l1.5 1.5"/>',
  grenade: '<circle cx="12" cy="14.5" r="5.5"/><path d="M10 7.5h4v2h-4zM12 5.5v2M15 8l2.5-2.5"/>',
  heavy: '<path d="M3 10.5h12v2.5H3zM3 15h12v2.5H3zM15 11.7h5M15 16.2h5M6.5 17.5v3"/>',
  crosshair: '<circle cx="12" cy="12" r="7"/><path d="M12 1.5v5M12 17.5v5M1.5 12h5M17.5 12h5"/>',
};

interface GlyphProps {
  op: Pick<Operative, "icon" | "accent" | "photo">;
  className?: string;
}

export function Glyph({ op, className }: GlyphProps) {
  if (op.photo) {
    return <img src={op.photo} alt="" />;
  }
  return (
    <svg
      className={`glyph ${className ?? ""}`}
      viewBox="0 0 24 24"
      fill="none"
      stroke={op.accent || "#8FA5B5"}
      strokeWidth={1.6}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      dangerouslySetInnerHTML={{ __html: ICONS[op.icon] || ICONS.rifle }}
    />
  );
}

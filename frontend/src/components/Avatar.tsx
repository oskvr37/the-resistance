import { useMemo } from "react";
import { createAvatar } from "@dicebear/core";
import * as adventurer from "@dicebear/adventurer";
import { StyleOptions } from "@dicebear/core";

export default function Avatar({ seed, options }: { seed: string, options: Partial<StyleOptions<adventurer.Options>> }) {
  const avatar = useMemo(() => {
    return createAvatar(adventurer, {
      size: 96,
      seed,
      ...options
    }).toDataUri();
  }, [seed, options]);

  return <img src={avatar} className="rounded-full" alt="Avatar" />;
}

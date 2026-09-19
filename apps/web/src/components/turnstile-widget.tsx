"use client";

import { Turnstile, type TurnstileInstance } from "@marsidev/react-turnstile";
import { useRef } from "react";

interface TurnstileWidgetProps {
  siteKey?: string;
  onSuccess?: (token: string) => void;
  onError?: (error: unknown) => void;
  onExpire?: () => void;
  className?: string;
}

export function TurnstileWidget({
  siteKey = process.env.NEXT_PUBLIC_TURNSTILE_SITE_KEY || "1x00000000000000000000AA",
  onSuccess,
  onError,
  onExpire,
  className,
}: TurnstileWidgetProps) {
  const turnstileRef = useRef<TurnstileInstance>(null);

  return (
    <div className={className}>
      <Turnstile
        ref={turnstileRef}
        siteKey={siteKey}
        onSuccess={onSuccess}
        onError={onError}
        onExpire={onExpire}
        options={{
          theme: "auto",
        }}
      />
    </div>
  );
}

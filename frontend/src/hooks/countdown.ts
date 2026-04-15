import { useEffect, useMemo, useState } from "react";

export const useCountdown = (targetTimestamp: number) => {
  // We use a function to get the current float value
  const getPrecisionTime = useMemo(
    () => () => {
      const now = Date.now() / 1000;
      const diff = targetTimestamp - now;
      return Math.max(0, diff);
    },
    [targetTimestamp],
  );

  const [timeLeft, setTimeLeft] = useState(getPrecisionTime());

  useEffect(() => {
    const timer = setInterval(() => {
      const remaining = getPrecisionTime();
      setTimeLeft(remaining);

      if (remaining <= 0) {
        clearInterval(timer);
      }
    }, 50);

    return () => clearInterval(timer);
  }, [targetTimestamp, getPrecisionTime]);

  return timeLeft;
};

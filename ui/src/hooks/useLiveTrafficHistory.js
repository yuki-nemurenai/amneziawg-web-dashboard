import { useEffect, useRef, useState } from 'react';

const MAX_POINTS = 36;

export default function useLiveTrafficHistory(status) {
  const [history, setHistory] = useState([]);
  const lastSampleRef = useRef(null);

  useEffect(() => {
    if (!status) return;

    const sample = {
      timestamp: Math.floor(Date.now() / 1000),
      rx_bytes: status.total_rx_bytes ?? 0,
      tx_bytes: status.total_tx_bytes ?? 0,
    };

    const lastSample = lastSampleRef.current;
    if (
      lastSample &&
      lastSample.rx_bytes === sample.rx_bytes &&
      lastSample.tx_bytes === sample.tx_bytes
    ) {
      return;
    }

    lastSampleRef.current = sample;
    setHistory((prev) => [...prev, sample].slice(-MAX_POINTS));
  }, [status?.total_rx_bytes, status?.total_tx_bytes, status]);

  return history;
}

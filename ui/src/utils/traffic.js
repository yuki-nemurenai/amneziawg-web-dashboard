export function formatBytes(bytes, decimals = 1) {
  if (!bytes || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const k = 1024;
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), units.length - 1);
  const value = bytes / k ** i;
  return `${value >= 100 ? value.toFixed(0) : value.toFixed(decimals)} ${units[i]}`;
}

export function formatBytesRate(bytesPerInterval, intervalSec = 5) {
  if (!bytesPerInterval || bytesPerInterval <= 0) return '0 B/s';
  const rate = bytesPerInterval / intervalSec;
  return `${formatBytes(rate)}/s`;
}

export function formatChartTime(timestamp) {
  const date = new Date(timestamp * 1000);
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

export function buildTrafficSeries(history = []) {
  if (history.length === 0) return [];

  const points = [];
  for (let i = 1; i < history.length; i += 1) {
    const prev = history[i - 1];
    const curr = history[i];
    const download = Math.max(0, curr.rx_bytes - prev.rx_bytes);
    const upload = Math.max(0, curr.tx_bytes - prev.tx_bytes);
    const intervalSec = Math.max(1, curr.timestamp - prev.timestamp);

    points.push({
      index: i - 1,
      timestamp: curr.timestamp,
      label: formatChartTime(curr.timestamp),
      download,
      upload,
      total: download + upload,
      downloadRate: download / intervalSec,
      uploadRate: upload / intervalSec,
      downloadFormatted: formatBytes(download),
      uploadFormatted: formatBytes(upload),
      downloadRateFormatted: formatBytesRate(download, intervalSec),
      uploadRateFormatted: formatBytesRate(upload, intervalSec),
    });
  }

  return points;
}

export function mergeTrafficHistory(serverHistory = [], liveHistory = []) {
  const byTimestamp = new Map();

  serverHistory.forEach((point) => {
    byTimestamp.set(point.timestamp, point);
  });

  liveHistory.forEach((point) => {
    byTimestamp.set(point.timestamp, point);
  });

  return Array.from(byTimestamp.values()).sort((a, b) => a.timestamp - b.timestamp);
}

export function generatePlaceholderSeries(count = 12) {
  const now = Math.floor(Date.now() / 1000);
  return Array.from({ length: count }, (_, i) => ({
    index: i,
    label: formatChartTime(now - (count - i) * 300),
    download: 0,
    upload: 0,
    total: 0,
    downloadFormatted: '0 B',
    uploadFormatted: '0 B',
    downloadRateFormatted: '0 B/s',
    uploadRateFormatted: '0 B/s',
  }));
}

export function buildPeersTrafficSeries(peerHistoryMap = {}, serverConfigPeers = []) {
  if (!peerHistoryMap || Object.keys(peerHistoryMap).length === 0) return { data: [], peers: [] };

  // Map public keys to peer names for readability
  const pubKeyToName = {};
  serverConfigPeers.forEach((p) => {
    pubKeyToName[p.public_key] = p.name;
  });

  const allTimestamps = new Set();
  Object.values(peerHistoryMap).forEach((history) => {
    history.forEach((point) => allTimestamps.add(point.timestamp));
  });

  const sortedTimestamps = Array.from(allTimestamps).sort((a, b) => a - b);
  const data = [];
  const activePeers = new Set();

  for (let i = 1; i < sortedTimestamps.length; i += 1) {
    const prevTs = sortedTimestamps[i - 1];
    const currTs = sortedTimestamps[i];
    const intervalSec = Math.max(1, currTs - prevTs);

    const pointData = {
      timestamp: currTs,
      label: formatChartTime(currTs),
    };

    Object.entries(peerHistoryMap).forEach(([pubKey, history]) => {
      const prev = history.find((p) => p.timestamp === prevTs);
      const curr = history.find((p) => p.timestamp === currTs);
      const peerName = pubKeyToName[pubKey] || pubKey.substring(0, 8);

      if (prev && curr) {
        const download = Math.max(0, curr.rx_bytes - prev.rx_bytes);
        const upload = Math.max(0, curr.tx_bytes - prev.tx_bytes);
        
        pointData[`${peerName}_download`] = download;
        pointData[`${peerName}_upload`] = upload;
        pointData[`${peerName}_total`] = download + upload;

        if (download > 0 || upload > 0) {
          activePeers.add(peerName);
        }
      } else {
        pointData[`${peerName}_download`] = 0;
        pointData[`${peerName}_upload`] = 0;
        pointData[`${peerName}_total`] = 0;
      }
    });

    data.push(pointData);
  }

  return { data, peers: Array.from(activePeers) };
}


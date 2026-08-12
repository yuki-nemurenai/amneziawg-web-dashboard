import React from 'react';
import { Area, AreaChart, ResponsiveContainer } from 'recharts';

export default function Sparkline({ data, dataKey, color, gradientId }) {
  if (!data?.length) {
    return <div className="mt-4 h-12 rounded-md bg-slate-50 dark:bg-white/[0.03]" />;
  }

  return (
    <div className="mt-4 h-12">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 4, right: 0, left: 0, bottom: 0 }}>
          <defs>
            <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={color} stopOpacity={0.35} />
              <stop offset="100%" stopColor={color} stopOpacity={0.02} />
            </linearGradient>
          </defs>
          <Area
            type="monotone"
            dataKey={dataKey}
            stroke={color}
            strokeWidth={2}
            fill={`url(#${gradientId})`}
            isAnimationActive
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts";

interface SpeedGraphProps {
  data: { time: number; up: number; down: number }[];
}

function formatSpeed(bytesPerSec: number): string {
  if (bytesPerSec === 0) return "0";
  const kbps = bytesPerSec / 1024;
  if (kbps < 1024) return `${kbps.toFixed(0)} KB/s`;
  return `${(kbps / 1024).toFixed(1)} MB/s`;
}

export default function SpeedGraph({ data }: SpeedGraphProps) {
  if (data.length === 0) {
    return (
      <div className="h-full flex items-center justify-center text-[var(--text-muted)] text-sm">
        Нет данных о скорости
      </div>
    );
  }

  const minDown = Math.min(...data.map((d) => d.down));
  const maxDown = Math.max(...data.map((d) => d.down));
  const minUp = Math.min(...data.map((d) => d.up));
  const maxUp = Math.max(...data.map((d) => d.up));
  const yMax = Math.max(maxDown, maxUp, 1) * 1.2;

  return (
    <ResponsiveContainer width="100%" height="100%">
      <AreaChart data={data} margin={{ top: 5, right: 5, bottom: 5, left: 0 }}>
        <defs>
          <linearGradient id="downGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#10b981" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="upGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#7c3aed" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#7c3aed" stopOpacity={0} />
          </linearGradient>
        </defs>
        <XAxis dataKey="time" hide />
        <YAxis domain={[0, yMax]} hide />
        <Tooltip
          contentStyle={{
            backgroundColor: "#1a1a2e",
            border: "1px solid #2a2a40",
            borderRadius: "8px",
            fontSize: "12px",
          }}
          formatter={(value: number) => [formatSpeed(value), ""]}
          labelFormatter={() => ""}
        />
        <Area
          type="monotone"
          dataKey="down"
          stroke="#10b981"
          fill="url(#downGradient)"
          strokeWidth={1.5}
          name="↓ Скачать"
        />
        <Area
          type="monotone"
          dataKey="up"
          stroke="#7c3aed"
          fill="url(#upGradient)"
          strokeWidth={1.5}
          name="↑ Отправить"
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}

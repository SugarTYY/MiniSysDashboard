import {
  CategoryScale,
  Chart as ChartJS,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  Title,
  Tooltip,
  Filler,
  TimeScale,
  ChartOptions
} from 'chart.js';
import { Line } from 'react-chartjs-2';
import { MetricData } from '../lib/api';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
  TimeScale
);

interface MetricsChartProps {
  title: string;
  data: MetricData[];
  dataKeys: { key: keyof MetricData; label: string; color: string; fill?: boolean }[];
  yAxisUnit?: string;
  maxPoints?: number;
}

export const MetricsChart = ({ title, data, dataKeys, yAxisUnit }: MetricsChartProps) => {
  const chartData = {
    labels: data.map((d) => {
      const date = new Date(d.timestamp);
      // Smart formatting based on time range could be added here
      // For now, just a readable locale string
      return date.toLocaleString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        month: '2-digit',
        day: '2-digit',
      });
    }),
    datasets: dataKeys.map((k) => ({
      label: k.label,
      data: data.map((d) => d[k.key]),
      borderColor: k.color,
      backgroundColor: k.fill ? `${k.color}20` : k.color, // 20% opacity for fill
      tension: 0.3,
      pointRadius: 0,
      pointHoverRadius: 4,
      borderWidth: 2,
      fill: k.fill || false,
      spanGaps: true,
    })),
  };

  const options: ChartOptions<'line'> = {
    responsive: true,
    animation: { duration: 0 },
    interaction: {
      mode: 'index',
      intersect: false,
    },
    plugins: {
      legend: {
        position: 'top',
        labels: {
          usePointStyle: true,
          boxWidth: 8,
          font: {
            family: "'Inter', sans-serif",
            size: 12
          }
        },
      },
      title: {
        display: true,
        text: title,
        font: {
          size: 14,
          weight: 'normal'
        },
        padding: { bottom: 20 }
      },
      tooltip: {
        backgroundColor: 'rgba(255, 255, 255, 0.9)',
        titleColor: '#1f2937',
        bodyColor: '#4b5563',
        borderColor: '#e5e7eb',
        borderWidth: 1,
        padding: 10,
        callbacks: {
          label: function(context) {
            let label = context.dataset.label || '';
            if (label) {
              label += ': ';
            }
            if (context.parsed.y !== null) {
              label += context.parsed.y;
              if (yAxisUnit) {
                label += ` ${yAxisUnit}`;
              }
            }
            return label;
          }
        }
      }
    },
    scales: {
      y: {
        beginAtZero: true,
        title: {
          display: !!yAxisUnit,
          text: yAxisUnit,
        },
        grid: {
          color: '#f3f4f6',
        },
        ticks: {
          font: {
            size: 10
          }
        }
      },
      x: {
        grid: {
          display: false,
        },
        ticks: {
          maxTicksLimit: 6,
          maxRotation: 0,
          font: {
            size: 10
          }
        },
      },
    },
    maintainAspectRatio: false,
  };

  return (
    <div className="h-72 w-full bg-white p-5 rounded-xl shadow-sm border border-gray-100 hover:shadow-md transition-shadow duration-300">
      <Line options={options} data={chartData} />
    </div>
  );
};

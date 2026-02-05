import { useEffect, useState, useMemo } from 'react';
import { 
  Activity, 
  Database, 
  LayoutDashboard, 
  Clock, 
  Calendar,
  Server,
  RefreshCw,
  Cpu,
  HardDrive,
  Network
} from 'lucide-react';
import { MetricsChart } from './components/MetricsChart';
import { fetchHistoryMetrics, MetricData, useRealtimeMetrics } from './lib/api';
import { downsampleMetrics } from './lib/utils';

const TIME_RANGES = [
  { label: '最近2小时', value: 2 * 60 * 60 * 1000 },
  { label: '最近6小时', value: 6 * 60 * 60 * 1000 },
  { label: '最近12小时', value: 12 * 60 * 60 * 1000 },
  { label: '最近1天', value: 24 * 60 * 60 * 1000 },
  { label: '最近3天', value: 3 * 24 * 60 * 60 * 1000 },
  { label: '最近1周', value: 7 * 24 * 60 * 60 * 1000 },
];

function App() {
  const [mode, setMode] = useState<'realtime' | 'history'>('realtime');
  const [metricsHistory, setMetricsHistory] = useState<MetricData[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  
  // History Query State
  const [selectedRange, setSelectedRange] = useState<number | null>(null);
  const [customStartTime, setCustomStartTime] = useState(Date.now() - 3600 * 1000);
  const [customEndTime, setCustomEndTime] = useState(Date.now());

  // Realtime Hook
  const { data: realtimeMetric, connected } = useRealtimeMetrics(mode === 'realtime');

  // Accumulate Realtime Data
  const MAX_REALTIME_POINTS = 60;
  
  useEffect(() => {
    if (realtimeMetric && mode === 'realtime') {
      setMetricsHistory((prev) => {
        const newData = [...prev, realtimeMetric];
        if (newData.length > MAX_REALTIME_POINTS) {
          return newData.slice(newData.length - MAX_REALTIME_POINTS);
        }
        return newData;
      });
    }
  }, [realtimeMetric, mode]);

  const displayMetrics = useMemo(() => {
    if (mode === 'realtime') return metricsHistory;
    // 目标约 800 个点以保证流畅渲染，同时使用 LTTB 算法保留峰值特征
    // 使用 cpu_usage 作为权重参考，因为 CPU 负载的突发性最强
    return downsampleMetrics(metricsHistory, 800, (d) => d.cpu_usage);
  }, [metricsHistory, mode]);

  const loadHistoryData = async (start: number, end: number) => {
    setIsLoading(true);
    try {
      const data = await fetchHistoryMetrics({ start, end });
      setMetricsHistory(data);
    } catch (err) {
      console.error(err);
      alert('加载历史数据失败');
    } finally {
      setIsLoading(false);
    }
  };

  const handleRangeSelect = (rangeValue: number) => {
    setSelectedRange(rangeValue);
    const end = Date.now();
    const start = end - rangeValue;
    loadHistoryData(start, end);
  };

  const handleCustomQuery = () => {
    setSelectedRange(null);
    loadHistoryData(customStartTime, customEndTime);
  };

  const handleSwitchToRealtime = () => {
    setMode('realtime');
    setMetricsHistory([]);
    setSelectedRange(null);
  };

  const handleSwitchToHistory = () => {
    setMode('history');
    setMetricsHistory([]);
    // Default to 2 hours
    handleRangeSelect(TIME_RANGES[0].value);
  };

  return (
    <div className="min-h-screen bg-gray-50 text-gray-900 font-sans selection:bg-indigo-100 selection:text-indigo-900">
      {/* Header */}
      <header className="bg-white border-b border-gray-100 sticky top-0 z-20 shadow-sm backdrop-blur-md bg-opacity-90">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="bg-indigo-600 p-2 rounded-lg">
              <LayoutDashboard className="w-5 h-5 text-white" />
            </div>
            <h1 className="text-lg font-bold text-gray-900 tracking-tight">MiniSys 监控看板</h1>
          </div>
          
          <div className="flex items-center space-x-4">
            <div className={`flex items-center space-x-2 px-3 py-1.5 rounded-full text-xs font-medium transition-colors duration-300 ${
              connected ? 'bg-green-50 text-green-700 border border-green-200' : 'bg-gray-100 text-gray-600 border border-gray-200'
            }`}>
              <div className={`w-2 h-2 rounded-full ${connected ? 'bg-green-500 animate-pulse' : 'bg-gray-400'}`} />
              <span>{mode === 'realtime' ? (connected ? '实时连接中' : '连接中...') : '历史回溯模式'}</span>
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
        {/* Controls Section */}
        <div className="bg-white p-5 rounded-xl shadow-sm border border-gray-100 flex flex-col md:flex-row gap-6 justify-between items-start md:items-center">
          {/* Mode Switcher */}
          <div className="flex bg-gray-100 p-1 rounded-lg">
            <button
              onClick={handleSwitchToRealtime}
              className={`px-4 py-2 rounded-md text-sm font-medium transition-all duration-200 flex items-center ${
                mode === 'realtime' 
                  ? 'bg-white text-indigo-600 shadow-sm' 
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              <Activity className="w-4 h-4 mr-2" />
              实时监控
            </button>
            <button
              onClick={handleSwitchToHistory}
              className={`px-4 py-2 rounded-md text-sm font-medium transition-all duration-200 flex items-center ${
                mode === 'history' 
                  ? 'bg-white text-indigo-600 shadow-sm' 
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              <Database className="w-4 h-4 mr-2" />
              历史回溯
            </button>
          </div>

          {/* History Controls */}
          {mode === 'history' && (
            <div className="flex flex-col md:flex-row gap-4 w-full md:w-auto animate-in fade-in duration-300">
              <div className="flex flex-wrap gap-2">
                {TIME_RANGES.map((range) => (
                  <button
                    key={range.label}
                    onClick={() => handleRangeSelect(range.value)}
                    className={`px-3 py-1.5 text-xs font-medium rounded-md border transition-colors ${
                      selectedRange === range.value
                        ? 'bg-indigo-50 border-indigo-200 text-indigo-700'
                        : 'bg-white border-gray-200 text-gray-600 hover:bg-gray-50'
                    }`}
                  >
                    {range.label}
                  </button>
                ))}
              </div>
              
              <div className="flex flex-col sm:flex-row items-start sm:items-center gap-2 border-t md:border-t-0 md:border-l pt-3 md:pt-0 pl-0 md:pl-4 border-gray-200 md:ml-2 w-full md:w-auto">
                 <div className="flex flex-col sm:flex-row items-center gap-2 w-full sm:w-auto">
                   <div className="w-full sm:w-auto">
                     <span className="sm:hidden text-xs text-gray-500 mb-1 block">开始时间</span>
                     <input 
                       type="datetime-local" 
                       className="w-full sm:w-auto border border-gray-200 rounded px-2 py-1.5 text-xs focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none"
                       onChange={(e) => setCustomStartTime(new Date(e.target.value).getTime())}
                     />
                   </div>
                   
                   <span className="hidden sm:inline text-gray-400">-</span>
                   
                   <div className="w-full sm:w-auto">
                     <span className="sm:hidden text-xs text-gray-500 mb-1 block">结束时间</span>
                     <input 
                       type="datetime-local" 
                       className="w-full sm:w-auto border border-gray-200 rounded px-2 py-1.5 text-xs focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none"
                       onChange={(e) => setCustomEndTime(new Date(e.target.value).getTime())}
                     />
                   </div>
                 </div>
                 <button 
                   onClick={handleCustomQuery}
                   className="w-full sm:w-auto p-1.5 bg-gray-900 text-white rounded-md hover:bg-gray-800 transition-colors flex justify-center items-center mt-2 sm:mt-0"
                   title="查询"
                 >
                   <RefreshCw className="w-4 h-4" />
                   <span className="ml-2 sm:hidden text-xs">执行查询</span>
                 </button>
              </div>
            </div>
          )}
        </div>

        {/* Loading State */}
        {isLoading && (
          <div className="flex justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
          </div>
        )}

        {/* Charts Grid */}
        {!isLoading && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            
            {/* CPU */}
            <MetricsChart 
              title="CPU 使用率 & 负载" 
              data={displayMetrics} 
              dataKeys={[
                { key: 'cpu_usage', label: '使用率 (%)', color: 'rgb(79, 70, 229)', fill: false },
                { key: 'cpu_load', label: '负载', color: 'rgb(236, 72, 153)' }
              ]}
            />
            
            <MetricsChart 
               title="CPU 温度" 
               data={displayMetrics} 
               dataKeys={[
                 { key: 'cpu_temp', label: '温度 (°C)', color: 'rgb(239, 68, 68)', fill: false }
               ]}
               yAxisUnit="°C"
             />

            {/* Memory */}
            <MetricsChart 
              title="内存使用率" 
              data={displayMetrics} 
              dataKeys={[
                { key: 'memory_usage', label: '使用率 (%)', color: 'rgb(16, 185, 129)', fill: false }
              ]}
            />

            {/* Disk IO */}
            <MetricsChart 
              title="磁盘 I/O 速率" 
              data={displayMetrics} 
              dataKeys={[
                { key: 'disk_read', label: '读取 (MB/s)', color: 'rgb(59, 130, 246)' },
                { key: 'disk_write', label: '写入 (MB/s)', color: 'rgb(245, 158, 11)' }
              ]}
              yAxisUnit="MB/s"
            />

            {/* Network Traffic */}
            <MetricsChart 
              title="网络吞吐量" 
              data={displayMetrics} 
              dataKeys={[
                { key: 'network_up', label: '上传 (Mbps)', color: 'rgb(99, 102, 241)' },
                { key: 'network_down', label: '下载 (Mbps)', color: 'rgb(139, 92, 246)' }
              ]}
              yAxisUnit="Mbps"
            />
            
            {/* Network Connections */}
             <MetricsChart 
               title="网络连接数" 
               data={displayMetrics} 
               dataKeys={[
                 { key: 'network_connections', label: '连接数', color: 'rgb(107, 114, 128)', fill: false }
               ]}
             />

          </div>
        )}
      </main>
    </div>
  );
}

export default App;

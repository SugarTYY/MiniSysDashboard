import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// Largest-Triangle-Three-Buckets (LTTB) Downsampling Algorithm
// Preserves visual peaks and trends better than simple step sampling
export function downsampleMetrics<T extends { timestamp: number }>(
  data: T[], 
  targetPoints: number,
  getValue: (item: T) => number
): T[] {
  if (data.length <= targetPoints || targetPoints < 3) return data;

  const sampled: T[] = [];
  sampled.push(data[0]); // Always keep the first point

  const bucketSize = (data.length - 2) / (targetPoints - 2);
  let a = 0; // Index of the last selected point

  for (let i = 0; i < targetPoints - 2; i++) {
    // Current bucket range
    const bucketStart = Math.floor((i + 1) * bucketSize) + 1;
    const bucketEnd = Math.floor((i + 2) * bucketSize) + 1;
    
    // Next bucket range (for calculating average point C)
    const nextBucketStart = bucketEnd;
    const nextBucketEnd = Math.floor((i + 3) * bucketSize) + 1;
    
    // Calculate Point C (Average of next bucket)
    let cx = 0, cy = 0, count = 0;
    // Limit nextBucketEnd to data length
    const safeNextEnd = Math.min(nextBucketEnd, data.length);
    
    for (let j = nextBucketStart; j < safeNextEnd; j++) {
      cx += data[j].timestamp;
      cy += getValue(data[j]);
      count++;
    }
    
    if (count > 0) {
      cx /= count;
      cy /= count;
    } else {
      // Fallback if next bucket is empty (shouldn't happen with correct math)
      const last = data[data.length - 1];
      cx = last.timestamp;
      cy = getValue(last);
    }

    // Point A (Last selected point)
    const pointAx = data[a].timestamp;
    const pointAy = getValue(data[a]);

    // Find Point B in current bucket that maximizes triangle area (A, B, C)
    let maxArea = -1;
    let maxAreaIndex = bucketStart;
    
    const safeBucketEnd = Math.min(bucketEnd, data.length);

    for (let j = bucketStart; j < safeBucketEnd; j++) {
      const x = data[j].timestamp;
      const y = getValue(data[j]);

      // Triangle Area = 0.5 * |(Ax - Cx)(By - Ay) - (Ax - Bx)(Cy - Ay)|
      const area = Math.abs(
        (pointAx - cx) * (y - pointAy) - 
        (pointAx - x) * (cy - pointAy)
      );

      if (area > maxArea) {
        maxArea = area;
        maxAreaIndex = j;
      }
    }

    sampled.push(data[maxAreaIndex]);
    a = maxAreaIndex; // Update last selected index
  }

  sampled.push(data[data.length - 1]); // Always keep the last point

  return sampled;
}

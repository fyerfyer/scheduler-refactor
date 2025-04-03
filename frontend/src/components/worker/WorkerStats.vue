<template>
  <div class="worker-stats">
    <div class="mb-4 d-flex justify-content-between align-items-center">
      <h5 class="mb-0">Worker Statistics</h5>
      <button class="btn btn-outline-primary btn-sm" @click="loadStats">
        <i class="bi bi-arrow-clockwise"></i> Refresh
      </button>
    </div>

    <div v-if="loading" class="text-center py-4">
      <div class="spinner-border text-primary" role="status">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>

    <div v-else class="row">
      <!-- Worker Count Card -->
      <div class="col-md-3 mb-3">
        <div class="card h-100">
          <div class="card-body text-center">
            <h5 class="card-title">Total Workers</h5>
            <div class="d-flex justify-content-center">
              <div class="display-4">{{ stats.total || 0 }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Online Workers Card -->
      <div class="col-md-3 mb-3">
        <div class="card h-100">
          <div class="card-body text-center">
            <h5 class="card-title">Online</h5>
            <div class="d-flex justify-content-center align-items-center">
              <div class="display-4 text-success">{{ stats.online || 0 }}</div>
              <div class="ms-2 text-muted">
                ({{ calculatePercentage(stats.online, stats.total) }}%)
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Offline Workers Card -->
      <div class="col-md-3 mb-3">
        <div class="card h-100">
          <div class="card-body text-center">
            <h5 class="card-title">Offline</h5>
            <div class="d-flex justify-content-center align-items-center">
              <div class="display-4 text-danger">{{ stats.offline || 0 }}</div>
              <div class="ms-2 text-muted">
                ({{ calculatePercentage(stats.offline, stats.total) }}%)
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- System Load Card -->
      <div class="col-md-3 mb-3">
        <div class="card h-100">
          <div class="card-body">
            <h5 class="card-title text-center">Average Load</h5>
            <div class="mt-3">
              <div class="d-flex justify-content-between mb-1">
                <span>CPU:</span>
                <span>{{ (stats.avgCpuUsage * 100).toFixed(1) }}%</span>
              </div>
              <div class="progress mb-3" style="height: 8px;">
                <div
                  class="progress-bar"
                  :class="getCpuBarClass(stats.avgCpuUsage)"
                  :style="{ width: `${(stats.avgCpuUsage * 100).toFixed(1)}%` }">
                </div>
              </div>

              <div class="d-flex justify-content-between mb-1">
                <span>Memory:</span>
                <span>{{ (stats.avgMemUsage * 100).toFixed(1) }}%</span>
              </div>
              <div class="progress" style="height: 8px;">
                <div
                  class="progress-bar"
                  :class="getMemoryBarClass(stats.avgMemUsage)"
                  :style="{ width: `${(stats.avgMemUsage * 100).toFixed(1)}%` }">
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'WorkerStats',
  data() {
    return {
      stats: {
        total: 0,
        online: 0,
        offline: 0,
        avgCpuUsage: 0,
        avgMemUsage: 0
      },
      loading: true
    }
  },
  created() {
    this.loadStats();
  },
  methods: {
    loadStats() {
      this.loading = true;

      // In a real app, this would be replaced with an actual API call
      fetch('/api/v1/worker/stats')
        .then(response => response.json())
        .then(data => {
          if (data.code === 0) {
            this.stats = data.data || this.stats;
          } else {
            console.error('Failed to load worker stats:', data.message);
          }
          this.loading = false;
        })
        .catch(error => {
          console.error('Error loading worker stats:', error);
          this.loading = false;
        });
    },
    calculatePercentage(value, total) {
      if (!total) return 0;
      return Math.round((value / total) * 100);
    },
    getCpuBarClass(usage) {
      if (usage < 0.6) return 'bg-success';
      if (usage < 0.8) return 'bg-warning';
      return 'bg-danger';
    },
    getMemoryBarClass(usage) {
      if (usage < 0.7) return 'bg-success';
      if (usage < 0.9) return 'bg-warning';
      return 'bg-danger';
    }
  }
}
</script>
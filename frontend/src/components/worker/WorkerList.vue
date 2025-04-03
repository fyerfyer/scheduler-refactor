<template>
  <div class="worker-list">
    <div class="mb-4 d-flex justify-content-between align-items-center">
      <h5 class="mb-0">Worker Nodes</h5>
      <button class="btn btn-outline-primary" @click="loadWorkers">
        <i class="bi bi-arrow-clockwise"></i> Refresh
      </button>
    </div>

    <div class="table-responsive">
      <table class="table table-hover">
        <thead class="table-light">
          <tr>
            <th>IP Address</th>
            <th>Hostname</th>
            <th>Status</th>
            <th>CPU Usage</th>
            <th>Memory Usage</th>
            <th>Last Seen</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="6" class="text-center py-4">
              <div class="spinner-border text-primary" role="status">
                <span class="visually-hidden">Loading...</span>
              </div>
            </td>
          </tr>
          <tr v-else-if="workers.length === 0">
            <td colspan="6" class="text-center py-4">
              No worker nodes found.
            </td>
          </tr>
          <tr v-for="worker in workers" :key="worker.ip">
            <td>{{ worker.ip }}</td>
            <td>{{ worker.hostname }}</td>
            <td>
              <status-badge
                :status="worker.status"
                type="worker"
              />
            </td>
            <td>
              <div class="progress" style="height: 10px;">
                <div
                  class="progress-bar"
                  :class="getCpuBarClass(worker.cpuUsage)"
                  :style="{ width: `${(worker.cpuUsage * 100).toFixed(0)}%` }"
                  :aria-valuenow="(worker.cpuUsage * 100).toFixed(0)"
                  aria-valuemin="0"
                  aria-valuemax="100">
                </div>
              </div>
              <small>{{ (worker.cpuUsage * 100).toFixed(0) }}%</small>
            </td>
            <td>
              <div class="progress" style="height: 10px;">
                <div
                  class="progress-bar"
                  :class="getMemoryBarClass(worker.memUsage)"
                  :style="{ width: `${(worker.memUsage * 100).toFixed(0)}%` }"
                  :aria-valuenow="(worker.memUsage * 100).toFixed(0)"
                  aria-valuemin="0"
                  aria-valuemax="100">
                </div>
              </div>
              <small>{{ (worker.memUsage * 100).toFixed(0) }}%</small>
            </td>
            <td>{{ formatLastSeen(worker.lastSeen) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script>
import StatusBadge from '../common/StatusBadge.vue';

export default {
  name: 'WorkerList',
  components: {
    StatusBadge
  },
  data() {
    return {
      workers: [],
      loading: true
    }
  },
  created() {
    this.loadWorkers();
  },
  methods: {
    loadWorkers() {
      this.loading = true;

      // In a real app, this would be replaced with an actual API call
      fetch('/api/v1/worker/list')
        .then(response => response.json())
        .then(data => {
          if (data.code === 0) {
            this.workers = data.data || [];
          } else {
            console.error('Failed to load workers:', data.message);
          }
          this.loading = false;
        })
        .catch(error => {
          console.error('Error loading workers:', error);
          this.loading = false;
        });
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
    },
    formatLastSeen(timestamp) {
      if (!timestamp) return 'N/A';

      // Convert milliseconds to seconds if needed
      if (timestamp > 1000000000000) {
        timestamp = Math.floor(timestamp / 1000);
      }

      const now = Math.floor(Date.now() / 1000);
      const diff = now - timestamp;

      if (diff < 60) {
        return `${diff} seconds ago`;
      } else if (diff < 3600) {
        return `${Math.floor(diff / 60)} minutes ago`;
      } else if (diff < 86400) {
        return `${Math.floor(diff / 3600)} hours ago`;
      } else {
        const date = new Date(timestamp * 1000);
        return date.toLocaleString();
      }
    }
  }
}
</script>

<style scoped>
.progress {
  margin-bottom: 2px;
}
</style>
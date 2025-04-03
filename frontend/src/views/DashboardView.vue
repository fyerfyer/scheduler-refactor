<template>
  <div class="dashboard-container">
    <h2 class="mb-4">Dashboard</h2>

    <!-- Loading state -->
    <div v-if="loading" class="text-center py-5">
      <div class="spinner-border text-primary" role="status">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>

    <div v-else>
      <!-- Summary Stats Cards -->
      <div class="row mb-4">
        <!-- Jobs Card -->
        <div class="col-md-3 mb-3">
          <div class="card h-100">
            <div class="card-body">
              <h5 class="card-title">Total Jobs</h5>
              <div class="d-flex align-items-center">
                <i class="bi bi-list-task fs-1 me-3 text-primary"></i>
                <div class="display-4">{{ jobStats.total }}</div>
              </div>
              <div class="mt-2 text-muted small">
                <span class="text-success me-2">{{ jobStats.enabled }} Enabled</span>
                <span class="text-secondary">{{ jobStats.disabled }} Disabled</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Workers Card -->
        <div class="col-md-3 mb-3">
          <div class="card h-100">
            <div class="card-body">
              <h5 class="card-title">Workers</h5>
              <div class="d-flex align-items-center">
                <i class="bi bi-cpu fs-1 me-3 text-primary"></i>
                <div class="display-4">{{ workerStats.total }}</div>
              </div>
              <div class="mt-2 text-muted small">
                <span class="text-success me-2">{{ workerStats.online }} Online</span>
                <span class="text-danger">{{ workerStats.offline }} Offline</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Job Executions Card -->
        <div class="col-md-3 mb-3">
          <div class="card h-100">
            <div class="card-body">
              <h5 class="card-title">Job Executions (24h)</h5>
              <div class="d-flex align-items-center">
                <i class="bi bi-activity fs-1 me-3 text-primary"></i>
                <div class="display-4">{{ logStats.totalCount }}</div>
              </div>
              <div class="mt-2 text-muted small">
                <span class="text-success me-2">{{ logStats.successCount }} Success</span>
                <span class="text-danger me-2">{{ logStats.failCount }} Failed</span>
                <span class="text-warning">{{ logStats.timeoutCount }} Timeout</span>
              </div>
            </div>
          </div>
        </div>

        <!-- System Health Card -->
        <div class="col-md-3 mb-3">
          <div class="card h-100">
            <div class="card-body">
              <h5 class="card-title">System Load</h5>
              <div class="mt-2">
                <div class="d-flex justify-content-between mb-1">
                  <span>CPU:</span>
                  <span>{{ (workerStats.avgCpuUsage * 100).toFixed(1) }}%</span>
                </div>
                <div class="progress mb-3" style="height: 8px;">
                  <div
                    class="progress-bar"
                    :class="getCpuBarClass(workerStats.avgCpuUsage)"
                    :style="{ width: `${(workerStats.avgCpuUsage * 100).toFixed(1)}%` }">
                  </div>
                </div>

                <div class="d-flex justify-content-between mb-1">
                  <span>Memory:</span>
                  <span>{{ (workerStats.avgMemUsage * 100).toFixed(1) }}%</span>
                </div>
                <div class="progress" style="height: 8px;">
                  <div
                    class="progress-bar"
                    :class="getMemoryBarClass(workerStats.avgMemUsage)"
                    :style="{ width: `${(workerStats.avgMemUsage * 100).toFixed(1)}%` }">
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Recent Activity and Workers -->
      <div class="row mb-4">
        <!-- Recent Executions Table -->
        <div class="col-md-8 mb-3">
          <div class="card h-100">
            <div class="card-header">
              <h5 class="card-title mb-0">Recent Job Executions</h5>
            </div>
            <div class="card-body">
              <div class="table-responsive">
                <table class="table table-hover">
                  <thead>
                    <tr>
                      <th>Job</th>
                      <th>Status</th>
                      <th>Start Time</th>
                      <th>Duration</th>
                      <th>Worker</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-if="recentLogs.length === 0">
                      <td colspan="5" class="text-center py-3">No recent job executions</td>
                    </tr>
                    <tr v-for="log in recentLogs" :key="`${log.jobName}-${log.startTime}`">
                      <td>{{ log.jobName }}</td>
                      <td>
                        <status-badge
                          :status="getLogStatus(log)"
                          type="execution"
                        />
                      </td>
                      <td>{{ formatDate(log.startTime) }}</td>
                      <td>{{ formatDuration(log.startTime, log.endTime) }}</td>
                      <td>{{ log.workerIp }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div class="text-end mt-2">
                <router-link to="/logs" class="btn btn-sm btn-outline-primary">
                  View All Logs
                </router-link>
              </div>
            </div>
          </div>
        </div>

        <!-- Worker Nodes -->
        <div class="col-md-4 mb-3">
          <div class="card h-100">
            <div class="card-header">
              <h5 class="card-title mb-0">Worker Nodes</h5>
            </div>
            <div class="card-body">
              <div v-if="workers.length === 0" class="text-center py-3">
                No worker nodes available
              </div>
              <div v-else>
                <div v-for="worker in workers" :key="worker.ip" class="worker-card mb-3 p-2 border rounded">
                  <div class="d-flex justify-content-between align-items-center mb-1">
                    <div class="fw-bold">{{ worker.hostname }}</div>
                    <status-badge :status="worker.status" type="worker" />
                  </div>
                  <div class="text-muted small mb-1">{{ worker.ip }}</div>
                  <div class="d-flex justify-content-between small text-muted">
                    <span>CPU: {{ (worker.cpuUsage * 100).toFixed(0) }}%</span>
                    <span>Memory: {{ (worker.memUsage * 100).toFixed(0) }}%</span>
                    <span>Last seen: {{ formatLastSeen(worker.lastSeen) }}</span>
                  </div>
                </div>
                <div class="text-end mt-2">
                  <router-link to="/workers" class="btn btn-sm btn-outline-primary">
                    View All Workers
                  </router-link>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Job Status Overview -->
      <div class="row mb-4">
        <div class="col-12">
          <div class="card">
            <div class="card-header">
              <h5 class="card-title mb-0">Job Status Overview</h5>
            </div>
            <div class="card-body">
              <div class="table-responsive">
                <table class="table table-hover">
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Schedule</th>
                      <th>Status</th>
                      <th>Last Run</th>
                      <th>Next Run</th>
                      <th class="text-center">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-if="jobs.length === 0">
                      <td colspan="6" class="text-center py-3">No jobs available</td>
                    </tr>
                    <tr v-for="job in jobs.slice(0, 5)" :key="job.name">
                      <td>
                        <router-link :to="`/jobs/${job.name}`" class="fw-bold text-decoration-none">
                          {{ job.name }}
                        </router-link>
                      </td>
                      <td>{{ job.cronExpr }}</td>
                      <td>
                        <status-badge :status="job.disabled ? 'disabled' : 'enabled'" type="job" />
                        <status-badge v-if="runningJobs.includes(job.name)" status="running" type="job" class="ms-1" />
                      </td>
                      <td>{{ getLastRunTime(job.name) }}</td>
                      <td>{{ getNextRunTime(job.cronExpr) }}</td>
                      <td class="text-center">
                        <job-status-actions
                          :job="job"
                          :is-running="runningJobs.includes(job.name)"
                          @reload="loadJobs"
                        />
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div class="text-end mt-2">
                <router-link to="/jobs" class="btn btn-sm btn-outline-primary">
                  View All Jobs
                </router-link>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import StatusBadge from '../components/common/StatusBadge.vue';
import JobStatusActions from '../components/job/JobStatusActions.vue';
import { parseExpression } from 'cron-parser';

export default {
  name: 'DashboardView',
  components: {
    StatusBadge,
    JobStatusActions
  },
  data() {
    return {
      loading: true,
      jobs: [],
      workers: [],
      recentLogs: [],
      runningJobs: [],
      jobStats: {
        total: 0,
        enabled: 0,
        disabled: 0
      },
      workerStats: {
        total: 0,
        online: 0,
        offline: 0,
        avgCpuUsage: 0,
        avgMemUsage: 0
      },
      logStats: {
        totalCount: 0,
        successCount: 0,
        failCount: 0,
        timeoutCount: 0
      },
      lastRunTimes: {} // Store last run time for each job
    }
  },
  created() {
    this.loadDashboardData();
    // Poll for updates every 30 seconds
    this.pollInterval = setInterval(this.loadDashboardData, 30000);
  },
  beforeUnmount() {
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
    }
  },
  methods: {
    loadDashboardData() {
      this.loading = true;

      // Load all the data in parallel
      Promise.all([
        this.loadJobs(),
        this.loadWorkers(),
        this.loadWorkerStats(),
        this.loadRecentLogs(),
        this.loadRunningJobs()
      ]).then(() => {
        this.loading = false;
      }).catch(error => {
        console.error('Error loading dashboard data:', error);
        this.loading = false;
      });
    },
    loadJobs() {
      return fetch('/api/v1/job/list')
        .then(response => response.json())
        .then(data => {
          if (data.code === 0) {
            this.jobs = data.data || [];

            // Calculate job statistics
            this.jobStats.total = this.jobs.length;
            this.jobStats.enabled = this.jobs.filter(job => !job.disabled).length;
            this.jobStats.disabled = this.jobs.filter(job => job.disabled).length;
          }
        })
        .catch(error => console.error('Error loading jobs:', error));
    },
    loadWorkers() {
      return fetch('/api/v1/worker/list')
        .then(response => response.json())
        .then(data => {
          if (data.code === 0) {
            this.workers = data.data || [];
          }
        })
        .catch(error => console.error('Error loading workers:', error));
    },
    loadWorkerStats() {
      return fetch('/api/v1/worker/stats')
        .then(response => response.json())
        .then(data => {
          if (data.code === 0) {
            this.workerStats = data.data || this.workerStats;
          }
        })
        .catch(error => console.error('Error loading worker stats:', error));
    },
    loadRecentLogs() {
      return fetch('/api/v1/log/list?page=1&pageSize=5')
        .then(response => response.json())
        .then(data => {
          if (data.code === 0 && data.data) {
            this.recentLogs = data.data.logs || [];

            // Store last run time for each job
            this.recentLogs.forEach(log => {
              if (!this.lastRunTimes[log.jobName] || log.startTime > this.lastRunTimes[log.jobName]) {
                this.lastRunTimes[log.jobName] = log.startTime;
              }
            });

            // Get log statistics for last 24 hours
            this.getLogStats();
          }
        })
        .catch(error => console.error('Error loading logs:', error));
    },
    getLogStats() {
      // In a real app, you would have an API endpoint for this
      // This is just a mock implementation
      fetch('/api/v1/log/stats/all?days=1')
        .then(response => response.json())
        .then(data => {
          if (data.code === 0) {
            this.logStats = data.data || this.logStats;
          }
        })
        .catch(error => console.error('Error loading log stats:', error));
    },
    loadRunningJobs() {
      return fetch('/api/v1/job/running')
        .then(response => response.json())
        .then(data => {
          if (data.code === 0) {
            this.runningJobs = data.data || [];
          }
        })
        .catch(error => console.error('Error checking running jobs:', error));
    },
    getLastRunTime(jobName) {
      const timestamp = this.lastRunTimes[jobName];
      if (!timestamp) return 'Never';
      return this.formatDate(timestamp);
    },
    getNextRunTime(cronExpr) {
      try {
        const interval = parseExpression(cronExpr);
        const nextDate = interval.next().toDate();
        return nextDate.toLocaleString();
      } catch (e) {
        return 'Invalid schedule';
      }
    },
    formatDate(timestamp) {
      if (!timestamp) return 'N/A';
      const date = new Date(timestamp * 1000);
      return date.toLocaleString();
    },
    formatDuration(startTime, endTime) {
      if (!startTime || !endTime) return 'N/A';

      const durationSec = endTime - startTime;

      if (durationSec < 60) {
        return `${durationSec} sec`;
      } else if (durationSec < 3600) {
        const minutes = Math.floor(durationSec / 60);
        const seconds = durationSec % 60;
        return `${minutes} min ${seconds} sec`;
      } else {
        const hours = Math.floor(durationSec / 3600);
        const minutes = Math.floor((durationSec % 3600) / 60);
        return `${hours} hr ${minutes} min`;
      }
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
    },
    getLogStatus(log) {
      if (log.isTimeout) return 'timeout';
      if (log.exitCode === 0) return 'success';
      return 'error';
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

<style scoped>
.dashboard-container {
  padding: 20px;
}

.worker-card {
  transition: all 0.2s ease;
  background-color: #f8f9fa;
}

.worker-card:hover {
  background-color: #f0f0f0;
  transform: translateY(-2px);
}
</style>
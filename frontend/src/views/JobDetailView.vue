<template>
  <div class="container-fluid">
    <div class="row mb-4">
      <div class="col">
        <h2>Job Details: {{ jobName }}</h2>
        <nav aria-label="breadcrumb">
          <ol class="breadcrumb">
            <li class="breadcrumb-item"><router-link to="/">Dashboard</router-link></li>
            <li class="breadcrumb-item"><router-link to="/jobs">Jobs</router-link></li>
            <li class="breadcrumb-item active" aria-current="page">{{ jobName }}</li>
          </ol>
        </nav>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="text-center py-5">
      <div class="spinner-border text-primary" role="status">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="alert alert-danger">
      {{ error }}
    </div>

    <!-- Job Details -->
    <div v-else>
      <!-- Job Information Card -->
      <div class="row mb-4">
        <div class="col-12">
          <div class="card">
            <div class="card-header d-flex justify-content-between align-items-center">
              <h5 class="card-title mb-0">Job Information</h5>
              <div>
                <status-badge :status="job.disabled ? 'disabled' : 'enabled'" type="job" class="me-2" />
                <status-badge v-if="isRunning" status="running" type="job" />
              </div>
            </div>
            <div class="card-body">
              <div class="row">
                <div class="col-md-6">
                  <dl class="row">
                    <dt class="col-sm-4">Name:</dt>
                    <dd class="col-sm-8">{{ job.name }}</dd>

                    <dt class="col-sm-4">Command:</dt>
                    <dd class="col-sm-8">
                      <pre class="command-display">{{ job.command }}</pre>
                    </dd>

                    <dt class="col-sm-4">Schedule:</dt>
                    <dd class="col-sm-8">{{ job.cronExpr }}</dd>
                  </dl>
                </div>
                <div class="col-md-6">
                  <dl class="row">
                    <dt class="col-sm-4">Timeout:</dt>
                    <dd class="col-sm-8">{{ job.timeout }} seconds</dd>

                    <dt class="col-sm-4">Created:</dt>
                    <dd class="col-sm-8">{{ formatDate(job.createdAt) }}</dd>

                    <dt class="col-sm-4">Last Updated:</dt>
                    <dd class="col-sm-8">{{ formatDate(job.updatedAt) }}</dd>

                    <dt class="col-sm-4">Next Run:</dt>
                    <dd class="col-sm-8">{{ getNextRunTime(job.cronExpr) }}</dd>
                  </dl>
                </div>
              </div>
            </div>
            <div class="card-footer d-flex justify-content-between">
              <div>
                <router-link :to="`/jobs/edit/${job.name}`" class="btn btn-primary me-2">
                  <i class="bi bi-pencil"></i> Edit Job
                </router-link>
                <button class="btn btn-outline-danger" @click="confirmDelete">
                  <i class="bi bi-trash"></i> Delete Job
                </button>
              </div>
              <div>
                <button
                    class="btn"
                    :class="job.disabled ? 'btn-success' : 'btn-secondary'"
                    @click="toggleStatus"
                >
                  <i :class="job.disabled ? 'bi bi-play-fill' : 'bi bi-pause-fill'"></i>
                  {{ job.disabled ? 'Enable' : 'Disable' }}
                </button>
                <button
                    v-if="isRunning"
                    class="btn btn-danger ms-2"
                    @click="killJob"
                >
                  <i class="bi bi-x-circle"></i> Kill Job
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Recent Executions -->
      <div class="row mb-4">
        <div class="col-12">
          <div class="card">
            <div class="card-header d-flex justify-content-between align-items-center">
              <h5 class="card-title mb-0">Recent Executions</h5>
              <button class="btn btn-sm btn-outline-primary" @click="loadJobLogs">
                <i class="bi bi-arrow-clockwise"></i> Refresh
              </button>
            </div>
            <div class="card-body">
              <div class="table-responsive">
                <table class="table table-hover">
                  <thead class="table-light">
                  <tr>
                    <th>Status</th>
                    <th>Start Time</th>
                    <th>Duration</th>
                    <th>Worker</th>
                    <th>Exit Code</th>
                    <th class="text-center">Actions</th>
                  </tr>
                  </thead>
                  <tbody>
                  <tr v-if="loadingLogs">
                    <td colspan="6" class="text-center py-3">
                      <div class="spinner-border spinner-border-sm text-primary" role="status">
                        <span class="visually-hidden">Loading...</span>
                      </div>
                    </td>
                  </tr>
                  <tr v-else-if="jobLogs.length === 0">
                    <td colspan="6" class="text-center py-3">No execution logs found for this job.</td>
                  </tr>
                  <tr v-for="log in jobLogs" :key="`${log.jobName}-${log.startTime}`">
                    <td>
                      <status-badge
                          :status="getLogStatus(log)"
                          type="execution"
                      />
                    </td>
                    <td>{{ formatDate(log.startTime) }}</td>
                    <td>{{ formatDuration(log.startTime, log.endTime) }}</td>
                    <td>{{ log.workerIp }}</td>
                    <td>{{ log.exitCode }}</td>
                    <td class="text-center">
                      <button class="btn btn-sm btn-outline-primary" @click="viewLogDetails(log)">
                        <i class="bi bi-file-text"></i> Details
                      </button>
                    </td>
                  </tr>
                  </tbody>
                </table>
              </div>
              <div v-if="jobLogs.length > 0" class="text-end mt-3">
                <router-link :to="`/logs?jobName=${jobName}`" class="btn btn-sm btn-outline-primary">
                  View All Logs
                </router-link>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Job Statistics -->
      <div class="row" v-if="jobStats">
        <div class="col-12">
          <div class="card">
            <div class="card-header">
              <h5 class="card-title mb-0">Execution Statistics (Last 7 Days)</h5>
            </div>
            <div class="card-body">
              <div class="row">
                <div class="col-md-3 text-center">
                  <h2 class="text-primary">{{ jobStats.totalCount || 0 }}</h2>
                  <p class="text-muted">Total Executions</p>
                </div>
                <div class="col-md-3 text-center">
                  <h2 class="text-success">{{ jobStats.successCount || 0 }}</h2>
                  <p class="text-muted">Successful</p>
                </div>
                <div class="col-md-3 text-center">
                  <h2 class="text-danger">{{ jobStats.failCount || 0 }}</h2>
                  <p class="text-muted">Failed</p>
                </div>
                <div class="col-md-3 text-center">
                  <h2 class="text-warning">{{ jobStats.timeoutCount || 0 }}</h2>
                  <p class="text-muted">Timed Out</p>
                </div>
              </div>
              <div class="row mt-3" v-if="jobStats.totalCount > 0">
                <div class="col-12">
                  <p class="mb-1">Success Rate</p>
                  <div class="progress">
                    <div
                        class="progress-bar bg-success"
                        role="progressbar"
                        :style="`width: ${getSuccessRate()}%`"
                        :aria-valuenow="getSuccessRate()"
                        aria-valuemin="0"
                        aria-valuemax="100">
                      {{ getSuccessRate() }}%
                    </div>
                  </div>
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
import StatusBadge from '../components/common/StatusBadge.vue';
import { parseExpression } from 'cron-parser';

export default {
  name: 'JobDetailView',
  components: {
    StatusBadge
  },
  data() {
    return {
      jobName: '',
      job: null,
      jobLogs: [],
      jobStats: null,
      loading: true,
      loadingLogs: true,
      error: null,
      isRunning: false,
      pollingInterval: null
    }
  },
  created() {
    this.jobName = this.$route.params.name;
    this.loadJob();
    this.loadJobLogs();
    this.loadJobStats();

    // Poll for running status every 5 seconds
    this.pollingInterval = setInterval(() => {
      this.checkRunningStatus();
    }, 5000);
  },
  beforeUnmount() {
    if (this.pollingInterval) {
      clearInterval(this.pollingInterval);
    }
  },
  methods: {
    loadJob() {
      this.loading = true;
      this.error = null;

      fetch(`/api/v1/job/${this.jobName}`)
          .then(response => response.json())
          .then(data => {
            if (data.code === 0) {
              this.job = data.data;
              this.checkRunningStatus();
            } else {
              this.error = data.message || 'Failed to load job';
            }
            this.loading = false;
          })
          .catch(error => {
            console.error('Error loading job:', error);
            this.error = 'An error occurred while loading job data';
            this.loading = false;
          });
    },
    loadJobLogs() {
      this.loadingLogs = true;

      fetch(`/api/v1/log/list?jobName=${this.jobName}&page=1&pageSize=5`)
          .then(response => response.json())
          .then(data => {
            if (data.code === 0 && data.data) {
              this.jobLogs = data.data.logs || [];
            } else {
              this.jobLogs = [];
            }
            this.loadingLogs = false;
          })
          .catch(error => {
            console.error('Error loading job logs:', error);
            this.jobLogs = [];
            this.loadingLogs = false;
          });
    },
    loadJobStats() {
      fetch(`/api/v1/log/stats/${this.jobName}?days=7`)
          .then(response => response.json())
          .then(data => {
            if (data.code === 0) {
              this.jobStats = data.data;
            }
          })
          .catch(error => {
            console.error('Error loading job stats:', error);
          });
    },
    checkRunningStatus() {
      fetch('/api/v1/job/running')
          .then(response => response.json())
          .then(data => {
            if (data.code === 0) {
              this.isRunning = (data.data || []).includes(this.jobName);
            }
          })
          .catch(error => {
            console.error('Error checking running jobs:', error);
          });
    },
    toggleStatus() {
      const endpoint = this.job.disabled ?
          `/api/v1/job/enable/${this.jobName}` :
          `/api/v1/job/disable/${this.jobName}`;

      fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      })
          .then(response => response.json())
          .then(data => {
            if (data.code === 0) {
              this.job.disabled = !this.job.disabled;
            } else {
              alert(`Failed to ${this.job.disabled ? 'enable' : 'disable'} job: ${data.message}`);
            }
          })
          .catch(error => {
            console.error('Error toggling job status:', error);
            alert('An error occurred while updating the job status');
          });
    },
    killJob() {
      if (!confirm(`Are you sure you want to kill the running job "${this.jobName}"?`)) {
        return;
      }

      fetch(`/api/v1/job/kill/${this.jobName}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      })
          .then(response => response.json())
          .then(data => {
            if (data.code === 0) {
              alert('Job kill signal sent successfully');
              this.checkRunningStatus();
            } else {
              alert(`Failed to kill job: ${data.message}`);
            }
          })
          .catch(error => {
            console.error('Error killing job:', error);
            alert('An error occurred while killing the job');
          });
    },
    confirmDelete() {
      if (!confirm(`Are you sure you want to delete the job "${this.jobName}"? This action cannot be undone.`)) {
        return;
      }

      fetch(`/api/v1/job/${this.jobName}`, {
        method: 'DELETE'
      })
          .then(response => response.json())
          .then(data => {
            if (data.code === 0) {
              alert('Job deleted successfully');
              this.$router.push('/jobs');
            } else {
              alert(`Failed to delete job: ${data.message}`);
            }
          })
          .catch(error => {
            console.error('Error deleting job:', error);
            alert('An error occurred while deleting the job');
          });
    },
    viewLogDetails(log) {
      this.$router.push(`/logs/${log.jobName}/${log.startTime}`);
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
    getLogStatus(log) {
      if (log.isTimeout) return 'timeout';
      if (log.exitCode === 0) return 'success';
      return 'error';
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
    getSuccessRate() {
      if (!this.jobStats || this.jobStats.totalCount === 0) return 0;
      const rate = (this.jobStats.successCount / this.jobStats.totalCount) * 100;
      return Math.round(rate);
    }
  }
}
</script>

<style scoped>
.breadcrumb {
  background-color: transparent;
  padding: 0;
  margin-bottom: 0;
}

.card {
  border-radius: 0.375rem;
  box-shadow: 0 0.125rem 0.25rem rgba(0, 0, 0, 0.075);
  margin-bottom: 1.5rem;
}

.card-header {
  border-bottom: 1px solid rgba(0, 0, 0, 0.125);
  padding: 1rem 1.25rem;
}

.card-body {
  padding: 1.25rem;
}

.card-footer {
  border-top: 1px solid rgba(0, 0, 0, 0.125);
  padding: 1rem 1.25rem;
  background-color: #fff;
}

.command-display {
  background-color: #f8f9fa;
  padding: 0.5rem;
  border-radius: 0.25rem;
  font-family: SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 0.875rem;
  white-space: pre-wrap;
  word-wrap: break-word;
  margin-bottom: 0;
  max-height: 100px;
  overflow-y: auto;
}

dl {
  margin-bottom: 0;
}

dt {
  font-weight: 500;
}

.progress {
  height: 1.5rem;
}

.progress-bar {
  font-weight: 500;
}
</style>
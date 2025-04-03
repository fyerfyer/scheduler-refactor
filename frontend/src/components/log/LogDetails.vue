<template>
  <div class="log-details">
    <div v-if="loading" class="text-center py-5">
      <div class="spinner-border text-primary" role="status">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>

    <div v-else-if="error" class="alert alert-danger">
      {{ error }}
    </div>

    <div v-else>
      <div class="card mb-4">
        <div class="card-header d-flex justify-content-between align-items-center">
          <h5 class="mb-0">Job Execution Details</h5>
          <status-badge
            :status="getLogStatus(log)"
            type="execution"
          />
        </div>
        <div class="card-body">
          <div class="row mb-3">
            <div class="col-md-6">
              <dl class="row mb-0">
                <dt class="col-sm-4">Job Name:</dt>
                <dd class="col-sm-8">{{ log?.jobName }}</dd>

                <dt class="col-sm-4">Start Time:</dt>
                <dd class="col-sm-8">{{ formatDate(log?.startTime) }}</dd>

                <dt class="col-sm-4">End Time:</dt>
                <dd class="col-sm-8">{{ formatDate(log?.endTime) }}</dd>
              </dl>
            </div>
            <div class="col-md-6">
              <dl class="row mb-0">
                <dt class="col-sm-4">Duration:</dt>
                <dd class="col-sm-8">{{ formatDuration(log?.startTime, log?.endTime) }}</dd>

                <dt class="col-sm-4">Exit Code:</dt>
                <dd class="col-sm-8">{{ log?.exitCode ?? 'N/A' }}</dd>

                <dt class="col-sm-4">Worker:</dt>
                <dd class="col-sm-8">{{ log?.workerIP || 'N/A' }}</dd>
              </dl>
            </div>
          </div>

          <h6 class="card-subtitle mb-2 text-muted">Command</h6>
          <div class="command-box mb-3">
            <pre><code>{{ log?.command || 'No command available' }}</code></pre>
          </div>

          <h6 class="card-subtitle mb-2 text-muted">Output</h6>
          <div class="command-box mb-3" v-if="log?.output">
            <pre><code>{{ log.output }}</code></pre>
          </div>
          <div v-else class="alert alert-info">No output recorded</div>
        </div>
        <div class="card-footer text-end">
          <button class="btn btn-primary me-2" @click="viewJobDetails">
            <i class="bi bi-info-circle"></i> View Job
          </button>
          <button class="btn btn-secondary" @click="goBack">
            <i class="bi bi-arrow-left"></i> Back to Logs
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import StatusBadge from '../common/StatusBadge.vue';

export default {
  name: 'LogDetails',
  components: {
    StatusBadge
  },
  props: {
    jobName: {
      type: String,
      required: false
    },
    timestamp: {
      type: [String, Number],
      required: false
    }
  },
  data() {
    return {
      log: null,
      loading: true,
      error: null
    }
  },
  created() {
    if (this.jobName && this.timestamp) {
      // Need to find specific log by timestamp
      this.findLogByTimestamp(this.jobName, this.timestamp);
    } else if (this.jobName) {
      // Load the latest log for the job
      this.loadLog(this.jobName);
    }
  },
  methods: {
    findLogByTimestamp(jobName, timestamp) {
      this.loading = true;
      this.error = null;

      // First, get all logs for this job
      const params = new URLSearchParams({
        jobName: jobName,
        pageSize: 100  // Set a large page size to increase chances of finding the log
      });

      fetch(`/api/v1/log/list?${params.toString()}`)
        .then(response => response.json())
        .then(data => {
          if (data.code === 0 && data.data?.logs) {
            // Find the specific log with the matching timestamp
            const specificLog = data.data.logs.find(log =>
              log.startTime.toString() === timestamp.toString()
            );

            if (specificLog) {
              this.log = specificLog;
            } else {
              this.error = 'Log not found with the specified timestamp';
            }
          } else {
            this.error = data.message || 'Failed to load logs';
          }
          this.loading = false;
        })
        .catch(error => {
          console.error('Error loading logs:', error);
          this.error = 'An error occurred while loading logs';
          this.loading = false;
        });
    },
    loadLog(jobName) {
      this.loading = true;
      this.error = null;

      fetch(`/api/v1/log/${jobName}`)
        .then(response => response.json())
        .then(data => {
          if (data.code === 0) {
            this.log = data.data;
          } else {
            this.error = data.message || 'Failed to load log details';
          }
          this.loading = false;
        })
        .catch(error => {
          console.error('Error loading log details:', error);
          this.error = 'An error occurred while loading log details';
          this.loading = false;
        });
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
      if (!log) return 'default';
      if (log.isTimeout) return 'timeout';
      if (log.exitCode === 0) return 'success';
      return 'error';
    },
    goBack() {
      // If using Vue Router, navigate back
      this.$router.back();
    },
    viewJobDetails() {
      if (this.log && this.log.jobName) {
        this.$router.push(`/jobs/${this.log.jobName}`);
      }
    }
  }
}
</script>

<style scoped>
.command-box, .output-box, .error-box {
  background-color: #f8f9fa;
  border-radius: 0.25rem;
  padding: 0.75rem;
  overflow-x: auto;
}

pre {
  white-space: pre-wrap;
  word-wrap: break-word;
  max-height: 300px;
  overflow-y: auto;
}

pre code {
  font-family: SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 0.875rem;
}

.error-box {
  background-color: rgba(220, 53, 69, 0.1);
}
</style>
<template>
  <div class="container-fluid">
    <div class="row mb-4">
      <div class="col">
        <h2>Execution Logs</h2>
        <nav aria-label="breadcrumb">
          <ol class="breadcrumb">
            <li class="breadcrumb-item"><router-link to="/">Dashboard</router-link></li>
            <li class="breadcrumb-item" :class="{'active': !timestamp}">
              <router-link v-if="timestamp" to="/logs">Logs</router-link>
              <span v-else>Logs</span>
            </li>
            <li v-if="timestamp" class="breadcrumb-item active" aria-current="page">
              Log Details
            </li>
          </ol>
        </nav>
      </div>
    </div>

    <div class="row">
      <div class="col">
        <div class="card">
          <div class="card-header bg-white">
            <h5 class="card-title mb-0">
              {{ timestamp ? 'Log Details' : 'Job Execution History' }}
            </h5>
          </div>
          <div class="card-body">
            <!-- Show LogDetails if timestamp is present, otherwise show LogList -->
            <log-details 
              v-if="timestamp" 
              :job-name="jobName" 
              :timestamp="timestamp"
            />
            <log-list 
              v-else 
              :selected-job="selectedJob" 
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import LogList from '../components/log/LogList.vue';
import LogDetails from '../components/log/LogDetails.vue';

export default {
  name: 'LogsView',
  components: {
    LogList,
    LogDetails
  },
  props: {
    jobName: {
      type: String,
      default: ''
    },
    timestamp: {
      type: [String, Number],
      default: null
    }
  },
  data() {
    return {
      selectedJob: ''
    }
  },
  created() {
    // Check if a specific job was requested in the URL but no timestamp
    // (only for the list view)
    if (!this.timestamp && this.$route.query.jobName) {
      this.selectedJob = this.$route.query.jobName;
    } else if (this.jobName) {
      this.selectedJob = this.jobName;
    }
  },
  watch: {
    // Watch for changes to the URL query parameters (for list view)
    '$route.query.jobName': function(newJobName) {
      if (!this.timestamp) {
        this.selectedJob = newJobName || '';
      }
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
</style>
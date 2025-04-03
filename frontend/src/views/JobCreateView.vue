<template>
  <div class="container-fluid">
    <div class="row mb-4">
      <div class="col">
        <h2>{{ isEditing ? 'Edit Job' : 'Create New Job' }}</h2>
        <nav aria-label="breadcrumb">
          <ol class="breadcrumb">
            <li class="breadcrumb-item"><router-link to="/">Dashboard</router-link></li>
            <li class="breadcrumb-item"><router-link to="/jobs">Jobs</router-link></li>
            <li class="breadcrumb-item active" aria-current="page">
              {{ isEditing ? 'Edit Job' : 'Create Job' }}
            </li>
          </ol>
        </nav>
      </div>
    </div>

    <div class="row">
      <div class="col-md-8">
        <div class="card">
          <div class="card-header bg-white">
            <h5 class="card-title mb-0">{{ isEditing ? 'Edit Job Settings' : 'Job Details' }}</h5>
          </div>
          <div class="card-body">
            <div v-if="loading" class="text-center py-5">
              <div class="spinner-border text-primary" role="status">
                <span class="visually-hidden">Loading...</span>
              </div>
            </div>
            <div v-else-if="error" class="alert alert-danger">
              {{ error }}
            </div>
            <div v-else>
              <job-form
                  :initial-job="job"
                  :is-editing="isEditing"
                  @submit="saveJob"
                  @cancel="cancelEdit"
              />
            </div>
          </div>
        </div>
      </div>

      <div class="col-md-4">
        <div class="card">
          <div class="card-header bg-white">
            <h5 class="card-title mb-0">Help</h5>
          </div>
          <div class="card-body">
            <h6>Job Name</h6>
            <p class="text-muted small mb-3">
              A unique identifier for the job. Cannot be changed after creation.
            </p>

            <h6>Command</h6>
            <p class="text-muted small mb-3">
              The shell command to execute. For example: <code>echo "Hello World"</code>
            </p>

            <h6>Cron Expression</h6>
            <p class="text-muted small mb-3">
              Defines the schedule for job execution using standard cron syntax.
            </p>
            <table class="table table-sm table-bordered text-muted small">
              <tbody>
                <tr>
                  <td><code>* * * * *</code></td>
                  <td>Every minute</td>
                </tr>
                <tr>
                  <td><code>0 * * * *</code></td>
                  <td>Every hour at minute 0</td>
                </tr>
                <tr>
                  <td><code>0 0 * * *</code></td>
                  <td>Every day at midnight</td>
                </tr>
                <tr>
                  <td><code>0 0 * * 0</code></td>
                  <td>Every Sunday at midnight</td>
                </tr>
                <!-- Other rows -->
              </tbody>
            </table>

            <h6>Timeout</h6>
            <p class="text-muted small mb-0">
              Maximum time in seconds allowed for job execution. Set to 0 for no timeout.
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import JobForm from '../components/job/JobForm.vue';

export default {
  name: 'JobCreateView',
  components: {
    JobForm
  },
  data() {
    return {
      job: {
        name: '',
        command: '',
        cronExpr: '',
        timeout: 60,
        disabled: false
      },
      loading: false,
      error: null,
      isEditing: false,
      jobSaved: false
    }
  },
  created() {
    // Check if we're editing an existing job
    const jobName = this.$route.params.name;
    if (jobName) {
      this.isEditing = true;
      this.loadJob(jobName);
    }
  },
  methods: {
    loadJob(jobName) {
      this.loading = true;
      this.error = null;

      fetch(`/api/v1/job/${jobName}`)
          .then(response => response.json())
          .then(data => {
            if (data.code === 0) {
              this.job = data.data;
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
    saveJob(jobData) {
      this.loading = true;
      this.error = null;

      fetch('/api/v1/job/save', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(jobData)
      })
          .then(response => response.json())
          .then(data => {
            if (data.code === 0) {
              // Job saved successfully
              this.jobSaved = true;

              // Show success message
              alert(`Job ${this.isEditing ? 'updated' : 'created'} successfully!`);

              // Navigate back to job details or jobs list
              if (this.isEditing) {
                this.$router.push(`/jobs/${jobData.name}`);
              } else {
                this.$router.push('/jobs');
              }
            } else {
              this.error = data.message || `Failed to ${this.isEditing ? 'update' : 'create'} job`;
              this.loading = false;
            }
          })
          .catch(error => {
            console.error('Error saving job:', error);
            this.error = `An error occurred while ${this.isEditing ? 'updating' : 'creating'} the job`;
            this.loading = false;
          });
    },
    cancelEdit() {
      if (this.isEditing) {
        this.$router.push(`/jobs/${this.job.name}`);
      } else {
        this.$router.push('/jobs');
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

code {
  background-color: #f8f9fa;
  padding: 0.2rem 0.4rem;
  border-radius: 0.2rem;
  font-size: 87.5%;
}
</style>
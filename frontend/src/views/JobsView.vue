<template>
  <div class="container-fluid">
    <div class="row mb-4">
      <div class="col">
        <h2>Job Management</h2>
        <nav aria-label="breadcrumb">
          <ol class="breadcrumb">
            <li class="breadcrumb-item"><router-link to="/">Dashboard</router-link></li>
            <li class="breadcrumb-item active" aria-current="page">Jobs</li>
          </ol>
        </nav>
      </div>
    </div>

    <div class="row">
      <div class="col">
        <div class="card">
          <div class="card-header bg-white">
            <h5 class="card-title mb-0">Jobs List</h5>
          </div>
          <div class="card-body">
            <job-list />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import JobList from '../components/job/JobList.vue';

export default {
  name: 'JobsView',
  components: {
    JobList
  },
  metaInfo: {
    title: 'Jobs Management'
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
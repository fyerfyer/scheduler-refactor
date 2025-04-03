<template>
  <nav class="navbar navbar-expand-lg navbar-dark bg-primary">
    <div class="container-fluid">
      <a class="navbar-brand" href="#">
        <i class="bi bi-clock"></i> Job Scheduler
      </a>
      <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarNav">
        <span class="navbar-toggler-icon"></span>
      </button>
      <div class="collapse navbar-collapse" id="navbarNav">
        <ul class="navbar-nav me-auto">
          <li class="nav-item">
            <router-link class="nav-link" to="/" active-class="active">Dashboard</router-link>
          </li>
          <li class="nav-item">
            <router-link class="nav-link" to="/jobs" active-class="active">Jobs</router-link>
          </li>
          <li class="nav-item">
            <router-link class="nav-link" to="/logs" active-class="active">Logs</router-link>
          </li>
          <li class="nav-item">
            <router-link class="nav-link" to="/workers" active-class="active">Workers</router-link>
          </li>
        </ul>
      </div>
    </div>
  </nav>
</template>

<script>
export default {
  name: 'Navbar'
}
</script>

<style scoped>
.navbar-brand i {
  margin-right: 8px;
}
</style>
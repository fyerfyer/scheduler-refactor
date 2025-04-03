<template>
  <div class="sidebar">
    <div class="sidebar-header">
      <h5>Menu</h5>
    </div>
    <ul class="nav flex-column">
      <li class="nav-item">
        <router-link to="/" class="nav-link" active-class="active">
          <i class="bi bi-speedometer2"></i> Dashboard
        </router-link>
      </li>
      <li class="nav-item">
        <router-link to="/jobs" class="nav-link" active-class="active">
          <i class="bi bi-list-task"></i> Jobs
        </router-link>
      </li>
      <li class="nav-item">
        <router-link to="/jobs/create" class="nav-link" active-class="active">
          <i class="bi bi-plus-circle"></i> Create Job
        </router-link>
      </li>
      <li class="nav-item">
        <router-link to="/logs" class="nav-link" active-class="active">
          <i class="bi bi-journal-text"></i> Logs
        </router-link>
      </li>
      <li class="nav-item">
        <router-link to="/workers" class="nav-link" active-class="active">
          <i class="bi bi-cpu"></i> Workers
        </router-link>
      </li>
    </ul>
  </div>
</template>

<script>
export default {
  name: 'Sidebar'
}
</script>

<style scoped>
.sidebar {
  background-color: #f8f9fa;
  height: 100%;
  padding: 20px 0;
  border-right: 1px solid #dee2e6;
}

.sidebar-header {
  padding: 0 20px 20px;
  border-bottom: 1px solid #dee2e6;
}

.nav-link {
  color: #495057;
  padding: 8px 20px;
}

.nav-link i {
  margin-right: 8px;
}

.nav-link.active {
  color: #0d6efd;
  background-color: rgba(13, 110, 253, 0.1);
}

.nav-link:hover {
  background-color: rgba(0, 0, 0, 0.05);
}
</style>
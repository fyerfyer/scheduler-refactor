<template>
  <div class="app-container">
    <Navbar />
    <div class="d-flex">
      <div class="sidebar-container d-none d-md-block">
        <Sidebar />
      </div>
      <div class="content-container">
        <router-view />
      </div>
    </div>
  </div>
</template>

<script>
import Navbar from './components/common/Navbar.vue'
import Sidebar from './components/common/Sidebar.vue'

export default {
  name: 'App',
  components: {
    Navbar,
    Sidebar
  }
}
</script>

<style>
/* Bootstrap 5 CSS */
@import 'bootstrap/dist/css/bootstrap.min.css';
/* Bootstrap Icons */
@import 'bootstrap-icons/font/bootstrap-icons.css';

body {
  background-color: #f8f9fa;
  font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
}

.app-container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.sidebar-container {
  width: 250px;
  min-height: calc(100vh - 56px);
}

.content-container {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  width: calc(100% - 250px);
}

@media (max-width: 768px) {
  .content-container {
    width: 100%;
  }
}

.card {
  box-shadow: 0 0.125rem 0.25rem rgba(0, 0, 0, 0.075);
  margin-bottom: 1.5rem;
}

.table th {
  font-weight: 500;
}

.btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.btn-icon i {
  margin-right: 5px;
}
</style>
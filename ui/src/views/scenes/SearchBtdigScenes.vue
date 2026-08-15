<template>
  <b-modal :active="isModalActive"           
           has-modal-card
           trap-focus
           aria-role="dialog"
           @cancel="close"
           aria-modal
           can-cancel>
    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">Search BTDIG: {{ scene.title }}</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>
      <div class="modal-card-body btdig-modal-body">
        <b-loading :is-full-page="false" v-model="isLoading"></b-loading>
        <iframe :src="btdigUrl" width="100%" height="100%" frameborder="0" sandbox="allow-scripts allow-same-origin allow-popups allow-forms" @load="isLoading = false"></iframe>
      </div>
    </div>
  </b-modal>
</template>

<script>
export default {
  name: 'SearchBtdigScenes',
  data () {
    return {
      isModalActive: true,
      isLoading: true,
      scene: {}
    }
  },
  mounted () {
    const item = Object.assign({}, this.$store.state.overlay.searchBtdigScenes.scene)    
    this.scene = item
  },
  computed: {
    btdigUrl () {
      if (!this.scene || !this.scene.title) return 'about:blank'
      let query = this.scene.title
      if (this.scene.site) {
        query = `${this.scene.site} ${this.scene.title}`
      }
      return `https://pt.btdig.com/search?order=0&q=${encodeURIComponent(query)}`
    }
  },
  methods: {
    close () {
      this.$store.commit('overlay/hideSearchBtdigScenes')
    }
  }
}
</script>

<style scoped>
.modal-card {
  width: 90vw;
  max-width: 1200px;
  height: 80vh;
  margin: 0 auto;
}
.btdig-modal-body {
  padding: 0;
  position: relative;
}
</style>

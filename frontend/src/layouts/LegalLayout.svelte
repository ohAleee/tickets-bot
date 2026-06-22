<Head/>

<div class="wrapper">
  <div class="super-container">
    <!-- TODO: Make this better :) -->
    <!-- {#if currentRoute.component !== Index}
      <LoadingScreen/>
    {/if} -->
    <NotifyModal/>
    <div class="content-container">
      <Route {currentRoute} {params}/>
    </div>
  </div>
</div>

<script>
    import {Route} from 'svelte-router-spa'
    import Head from '../includes/Head.svelte'
    import Sidebar from '../includes/Sidebar.svelte'
    import LoadingScreen from '../includes/LoadingScreen.svelte'
    import NotifyModal from '../includes/NotifyModal.svelte'
    import {loadingScreen} from "../js/stores"
    import Index from "../views/Index.svelte";

    export let currentRoute;
    export let params = {};
</script>

<style>
    body {
        padding: 0 !important;
    }

    .wrapper {
        display: flex;
        width: 100%;
        height: 100%;
        margin: 0 !important;
        padding: 0 !important;
    }

    .super-container {
        display: flex;
        width: 100%;
        height: 100%;
    }

    .content-container {
        display: flex;
        width: 100%;
        height: 100%;
    }

    .hide {
        visibility: hidden;
    }

    @media (max-width: 950px) {
        .wrapper {
            flex-direction: column;
        }
    }
</style>
export const CLUSTER_BEGIN = 'CLUSTER_BEGIN'
export const CLUSTER_SUCCESS = 'CLUSTER_SUCCESS'
export const CLUSTER_FAILURE = 'CLUSTER_FAILURE'

export const getCluster = () => (dispatch) => _getCluster(dispatch)

const _clusterBegin = () => ({type: CLUSTER_BEGIN})
const _clusterSuccess = (data) => ({type: CLUSTER_SUCCESS, payload: { data: data }})
const _clusterFailure = (error) => ({type: CLUSTER_FAILURE, payload: error})

const _getCluster = (dispatch) => {
  dispatch(_clusterBegin());
  return fetch('/api/v1/cluster',
    {
      method: "GET",
      headers: {'Content-Type': 'application/json'}
    })
    .then(response => {
      if (response.status >= 200 && response.status < 300) {
          return response
      } else {
          let error = new Error(response.statusText);
          error.response = response;
          throw error
      }
    })
    .then(response => {
      return response.json()
    })
    .then(response => {
          dispatch(_clusterSuccess(response));
    })
    .catch(error => {
        if(!error.response) {
          dispatch(_clusterFailure({ status: 500, statusText: error.message}));
        } else if(error.response.status !== 200) {
          dispatch(_clusterFailure({ status: error.response.status, statusText: error}));
        }
    });
}
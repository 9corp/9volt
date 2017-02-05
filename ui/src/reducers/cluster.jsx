import {createReducer} from '../utils';
import {CLUSTER_BEGIN,CLUSTER_SUCCESS,CLUSTER_FAILURE} from '../actions/cluster';

const initialState = {
    isFetching: false,
    data: null,
    statusText: ""
};

export default createReducer(initialState,{
    [CLUSTER_BEGIN]: (state,payload) => {
        return Object.assign({},state, {
            isFetching: true,
            statusText: "Retrieving clusters."
        });
    },
    [CLUSTER_SUCCESS]: (state,payload) => {
        return Object.assign({},state, {
            isFetching: true,
            data: payload.data,
            statusText: "Clusters retrieved."
        });
    },
    [CLUSTER_FAILURE]: (state,payload) => {
        return Object.assign({},state, {
            isFetching: true,
            statusText: `Clusters Error: ${payload.status} ${payload.statusText}`
        });
    }
});
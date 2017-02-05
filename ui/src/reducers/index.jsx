import {combineReducers} from 'redux';
import {routerReducer} from 'react-router-redux'
import cluster from './cluster';
import status from './status';
import events from './events';

export default combineReducers({
    cluster: cluster,
    status: status,
    events: events,
    routing: routerReducer
});

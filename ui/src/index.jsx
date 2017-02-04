import React from 'react';
import ReactDOM from 'react-dom';
import Root from './containers/Root';
import configureStore from './store/configureStore';
import { browserHistory } from 'react-router';
import { syncHistoryWithStore } from 'react-router-redux'

// import "./styles/bootstrap.min.css

const target = document.getElementById('root');
const store = configureStore(window.__INITIAL_STATE__,browserHistory);
const history = syncHistoryWithStore(browserHistory,store);
const node = <Root store={store} history={history} />;

ReactDOM.render(node,target);

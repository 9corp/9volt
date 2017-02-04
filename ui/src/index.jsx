import React from 'react';
import ReactDOM from 'react-dom';
import {Provider} from 'react-redux';
import { browserHistory, Router, Route, IndexRoute } from 'react-router';
import { syncHistoryWithStore } from 'react-router-redux'

import {setStore} from './store';
import {HomeView,StatusView,SettingsView} from './views';
import {App} from './app';

import '../semantic/dist/semantic.min.css';

const store = setStore(window.__INITIAL_STATE__,browserHistory);
const history = syncHistoryWithStore(browserHistory,store);
    
const Root = ({store,history}) => {
    return (
        <Provider store={store}>
            <Router history={history}>
                <Route component={App}>
                    <Route path="/ui">
                        <IndexRoute component={HomeView}/>
                        <Route path="/ui/Status" component={StatusView}/>
                        <Route path="/ui/Settings" component={SettingsView}/>
                    </Route>
                </Route>
            </Router>
        </Provider>
    );
}

ReactDOM.render(<Root store={store} history={history}/>,document.getElementById('root'));

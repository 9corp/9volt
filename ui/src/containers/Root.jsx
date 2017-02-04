import React from 'react';
import {Provider} from 'react-redux';
import {Router} from 'react-router';
import routes from '../routes';

export default class Root extends React.Component {

    static propTypes = {
        store: React.PropTypes.object.isRequired,
        history: React.PropTypes.object.isRequired,
    };

    render () {
        return (
            <div>
                <Provider store={this.props.store}>
                    <div>
                        <Router history={this.props.history}>
                            {routes}
                        </Router>
                    </div>
                </Provider>
            </div>
        );
    }
}

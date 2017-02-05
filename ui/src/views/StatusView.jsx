import React from 'react';
import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
import * as actionCreators from '../actions/status';
import {Header,Message,Button} from 'semantic-ui-react'
import JSONPretty from 'react-json-pretty';
import 'react-json-pretty/JSONPretty.adventure_time.styl';

class StatusView extends React.Component {

    componentWillMount() {
        this.props.actions.getStatus();
    }

    render() {
        const { data,statusText,actions } = this.props;

        return (
            <div>
                <Header as='h2'>Status View</Header>
                <Message>
                    <Message.Header>
                        Status
                    </Message.Header>
                    <p>
                        {statusText}
                    </p>
                </Message>
                <Button content='Refresh' icon='refresh' labelPosition='left' primary onClick={() => actions.getStatus()}/>
                <JSONPretty id="json-pretty" json={data}></JSONPretty>
            </div>
        );
    }
    
};

const mapStateToProps = (state) => ({
  data: state.status.data,
  isFetching: state.status.isFetching,
  statusText: state.status.statusText
});

const mapDispatchToProps = (dispatch) => ({
    actions: bindActionCreators(actionCreators, dispatch)
});

export default connect(mapStateToProps,mapDispatchToProps)(StatusView)

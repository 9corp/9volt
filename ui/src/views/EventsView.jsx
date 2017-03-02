import React from 'react';
import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
import * as actionCreators from '../actions/events';
import {Header,Message,Button, List} from 'semantic-ui-react'
import dateformat from 'dateformat';
import 'react-json-pretty/JSONPretty.adventure_time.styl';

class EventsView extends React.Component {

    componentWillMount() {
        this.props.actions.getEvents();
    }

    render() {
        const { data,statusText,actions } = this.props;

        return (
            <div style={styles.container}>
                <Header as='h2'>Events View</Header>
                <Message>
                    <Message.Header>
                        Status
                    </Message.Header>
                    <p>
                        {statusText}
                    </p>
                </Message>
                <Button content='Refresh' icon='refresh' labelPosition='left' primary onClick={() => actions.getEvents()}/>

                <List divided relaxed>
                    { data && Object.keys(data).map((key) => {
                        const item = data[key];
                        const eventTime = dateformat(Date.parse(item.timestamp),"dddd, mmmm dS, yyyy, h:MM:ss.l TT Z");

                        return (
                            <List.Item key={key}>
                                <List.Icon name='warning circle' color='red' size='large' verticalAlign='middle'/>
                                <List.Content>
                                    <List.Header as="h3">{key}</List.Header>
                                    <List.Description>
                                        {item.message}<br />
                                        <span style={styles.type}>{item.type}</span> -- 
                                        <span style={styles.id}> Member ID: {item.memberid}</span> -- 
                                        <span style={styles.time}> {eventTime}</span>
                                    </List.Description>
                                </List.Content>
                            </List.Item>
                        );
                    })

                    }
                </List>
            </div>
        );
    }
    
};

const mapStateToProps = (state) => ({
  data: state.events.data,
  isFetching: state.events.isFetching,
  statusText: state.events.statusText
});

const mapDispatchToProps = (dispatch) => ({
    actions: bindActionCreators(actionCreators, dispatch)
});

export default connect(mapStateToProps,mapDispatchToProps)(EventsView)

const styles = {
    container: {
        marginBottom:30
    },
    type:{
        color:'grey'
    },
    id: {
        color:'black',
        fontWeight:'bold'
    },
    time: {
        color:'#2185D0',
        fontWeight:'bold'
    }
}
import React from 'react';
import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
import * as actionCreators from '../actions/cluster';
import {Header,Message,Button,Card,Icon,Image} from 'semantic-ui-react'
import dateformat from 'dateformat';
import Image9 from '../assets/9volt-image-half.jpg'
import 'react-json-pretty/JSONPretty.adventure_time.styl';

class ClusterView extends React.Component {

    componentWillMount() {
        this.props.actions.getCluster();
    }

    render() {
        const { data,statusText,actions } = this.props;

        return (
            <div>
                <Header as='h2'>Cluster View</Header>
                <Message>
                    <Message.Header>
                        Status
                    </Message.Header>
                    <p>
                        {statusText}
                    </p>
                </Message>
                <Button content='Refresh' icon='refresh' labelPosition='left' primary onClick={() => actions.getCluster()}/>

                { data && data.Members && Object.keys(data.Members).map(key => {
                    const i = data.Members[key];
                    const updated = dateformat(Date.parse(i.LastUpdated),"dddd, mmmm dS, yyyy, h:MM:ss.l TT Z");
                    const isDirector = i.MemberID === data.Director.MemberID;

                    return (
                        <Card>
                            <Image src={Image9} />
                            <Card.Content>
                                <Card.Header>
                                    {i.MemberID}
                                </Card.Header>
                                <Card.Meta>
                                    <span>
                                    {i.Hostname}
                                    </span>
                                </Card.Meta>
                                <Card.Description>
                                    Address: {i.ListenAddress}
                                </Card.Description>
                            </Card.Content>
                            <Card.Content extra>
                                <Icon name='clock' />
                                {updated}
                            </Card.Content>
                            { isDirector && <Card.Content extra>
                                    <div style={styles.director}>
                                        <Icon name='pied piper hat' />
                                        Director!
                                    </div>
                                </Card.Content>
                            }
                           
                        </Card>
                    );
                })}
            </div>
        );
    }
    
};

const mapStateToProps = (state) => ({
  data: state.cluster.data,
  isFetching: state.cluster.isFetching,
  statusText: state.cluster.statusText
});

const mapDispatchToProps = (dispatch) => ({
    actions: bindActionCreators(actionCreators, dispatch)
});

export default connect(mapStateToProps,mapDispatchToProps)(ClusterView)

const styles = {
    director: {
        color: 'red'
    }
}
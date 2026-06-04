import React from 'react';
import { withRouter } from 'react-router-dom';
import { Switch, Route, Redirect } from 'react-router';
import { client as clientRoute } from 'App/routes';
import { CLIENT_TABS } from 'App/utils/routeUtils';
import { PANEL_SIZES } from 'App/constants/panelSizes';

import SessionsListingSettings from 'Components/Client/SessionsListingSettings';
// Replay-only build: keep legacy preference tabs commented for later.
// import Modules from 'Components/Client/Modules';
import ProfileSettings from './ProfileSettings';
// import Integrations from './Integrations';
import UserView from './Users/UsersView';
// import AuditView from './Audit/AuditView';
// import Billing from './Billing/Billing';
import Projects from './Projects';
// import CustomFields from './CustomFields';
// import Webhooks from './Webhooks';
// import Notifications from './Notifications';
// import Roles from './Roles';
// import ExportedVideosList from './ExportedVideos/ExportedVideosList';

class Client extends React.PureComponent {
  constructor(props) {
    super(props);
  }

  setTab = (tab) => {
    this.props.history.push(clientRoute(tab));
  };

  renderActiveTab = () => (
    <Switch>
      <Route
        exact
        strict
        path={clientRoute(CLIENT_TABS.PROFILE)}
        component={ProfileSettings}
      />
      <Route
        exact
        strict
        path={clientRoute(CLIENT_TABS.SESSION_SETTINGS)}
        component={SessionsListingSettings}
      />
      <Route
        exact
        strict
        path={clientRoute(CLIENT_TABS.INTEGRATIONS)}
        render={() => <Redirect to={clientRoute(CLIENT_TABS.PROFILE)} />}
      />
      <Route
        exact
        strict
        path={clientRoute(CLIENT_TABS.MANAGE_USERS)}
        component={UserView}
      />
      <Route
        exact
        strict
        path={clientRoute(CLIENT_TABS.SITES)}
        component={Projects}
      />
      <Route
        exact
        strict
        path={clientRoute(CLIENT_TABS.CUSTOM_FIELDS)}
        render={() => <Redirect to={clientRoute(CLIENT_TABS.PROFILE)} />}
      />
      <Route
        exact
        strict
        path={clientRoute(CLIENT_TABS.BILLING)}
        render={() => <Redirect to={clientRoute(CLIENT_TABS.PROFILE)} />}
      />
      <Route
        exact
        strict
        path={clientRoute(CLIENT_TABS.WEBHOOKS)}
        render={() => <Redirect to={clientRoute(CLIENT_TABS.PROFILE)} />}
      />
      <Route
        exact
        strict
        path={clientRoute(CLIENT_TABS.NOTIFICATIONS)}
        render={() => <Redirect to={clientRoute(CLIENT_TABS.PROFILE)} />}
      />
      <Route
        exact
        strict
        path={clientRoute(CLIENT_TABS.MANAGE_ROLES)}
        render={() => <Redirect to={clientRoute(CLIENT_TABS.PROFILE)} />}
      />
      <Route
        exact
        strict
        path={clientRoute(CLIENT_TABS.AUDIT)}
        render={() => <Redirect to={clientRoute(CLIENT_TABS.PROFILE)} />}
      />
      <Route
        exact
        strict
        path={clientRoute(CLIENT_TABS.MODULES)}
        render={() => <Redirect to={clientRoute(CLIENT_TABS.PROFILE)} />}
      />
      <Route
        exact
        strict
        path={clientRoute(CLIENT_TABS.VIDEOS)}
        render={() => <Redirect to={clientRoute(CLIENT_TABS.PROFILE)} />}
      />
      <Redirect to={clientRoute(CLIENT_TABS.PROFILE)} />
    </Switch>
  );

  render() {
    const {
      match: {
        params: { activeTab },
      },
    } = this.props;
    return (
      <div
        className="w-full mx-auto mb-8"
        style={{ maxWidth: PANEL_SIZES.maxWidth }}
      >
        {activeTab && this.renderActiveTab()}
      </div>
    );
  }
}

export default withRouter(Client);

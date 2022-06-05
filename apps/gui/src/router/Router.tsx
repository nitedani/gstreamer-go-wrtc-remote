import {
  BrowserRouter,
  Navigate,
  Route,
  Routes as Switch,
} from 'react-router-dom';
import { Home } from '../pages/home/home';

export const Routes = (): JSX.Element => {
  return (
    <BrowserRouter>
      <Switch>
        <Route path="/" element={<Home />} />
        <Route path="*" element={<Navigate to="/" />} />
      </Switch>
    </BrowserRouter>
  );
};

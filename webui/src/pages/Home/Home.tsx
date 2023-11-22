import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

const Home = () => {
    const navigate = useNavigate();

    useEffect(() => {
        navigate('/JobsDashboard', { replace: false });
    }, [navigate]);

    return null;
};
  

export default Home;
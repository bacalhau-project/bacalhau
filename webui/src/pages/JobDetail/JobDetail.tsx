import styles from "./JobDetail.module.scss";
import Layout from "../../layout/Layout";
import Container from "../../components/Container/Container";

const JobDetail: React.FC = () => {
  return (
    <Layout pageTitle="Job Detail">
      <div className={styles.jobDetail}>
        <div>
          <Container title={"Job Overview"}/>
          <Container title={"Execution Record"}/>
        </div>
        <div>
          <Container title={"Execution Details"}/>
          <Container title={"Standard Output"}/>
          <Container title={"Execution Logs"}/>
        </div>
        <div>
          <Container title={"Inputs"}/>
          <Container title={"Input"}/>
          <Container title={"Outputs"}/>
          <Container title={"Output"}/>
        </div>
      </div>
    </Layout>
  );
};

export default JobDetail;

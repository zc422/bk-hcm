import { defineComponent, ref, watch } from 'vue';
// import components
import { Tab } from 'bkui-vue';
import ListenerList from './listener-list';
import TargetGroupDetail from './target-group-detail';
import HealthCheckupPage from './health-checkup';
// import stores
import { useBusinessStore, useLoadBalancerStore } from '@/store';
// import hooks
import useActiveTab from '@/hooks/useActiveTab';
import './index.scss';

const { TabPanel } = Tab;

enum TabType {
  list = 'list',
  detail = 'detail',
  health = 'health',
}

export default defineComponent({
  name: 'SpecificTargetGroupManager',
  props: { id: String },
  setup(props) {
    const businessStore = useBusinessStore();
    const loadBalancerStore = useLoadBalancerStore();
    const tgDetail = ref({});
    const { activeTab, handleActiveTabChange } = useActiveTab(TabType.list);
    const tabList = [
      { name: TabType.list, label: '绑定的监听器', component: ListenerList },
      { name: TabType.detail, label: '基本信息', component: TargetGroupDetail },
      { name: TabType.health, label: '健康检查', component: HealthCheckupPage },
    ];

    const getTargetGroupDetail = async (id: string) => {
      const res = await businessStore.getTargetGroupDetail(id);
      tgDetail.value = res.data;
    };

    watch(
      () => props.id,
      (id) => {
        if (!id) return;
        // 目标组id状态保持
        loadBalancerStore.setTargetGroupId(id);
        getTargetGroupDetail(id);
      },
      {
        immediate: true,
      },
    );

    return () => (
      <div class='specific-target-group-manager'>
        <Tab
          class='manager-tab-wrap'
          v-model:active={activeTab.value}
          type='card-grid'
          onChange={handleActiveTabChange}>
          {tabList.map((tab) => (
            <TabPanel key={tab.name} name={tab.name} label={tab.label}>
              <div class='common-card-wrap'>
                {<tab.component detail={tgDetail.value} getTargetGroupDetail={getTargetGroupDetail} />}
              </div>
            </TabPanel>
          ))}
        </Tab>
      </div>
    );
  },
});

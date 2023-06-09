import { Button, Dialog, Message } from "bkui-vue";
import { PropType, computed, defineComponent, ref, watch } from "vue";
import "./index.scss";
import { usePreviousState } from "@/hooks/usePreviousState";
import { useRoute } from "vue-router";
import { useResourceStore } from "@/store";

export enum Operations {
  None = 'none',
  Open = 'start',
  Close = 'stop',
  Reboot = 'reboot',
  Recycle = 'destroy',
}

export const OperationsMap = {
  [Operations.Open]: "开机",
  [Operations.Close]: "关机",
  [Operations.Reboot]: "重启",
  [Operations.Recycle]: "回收",
};

export const HOST_SHUTDOWN_STATUS = [
  "TERMINATED",
  "PowerState/stopped",
  "SHUTOFF",
  "STOPPED",
  "STOPPING",
  "PowerState/stopping",
  "stopped",
];
export const HOST_RUNNING_STATUS = [
  "STAGING",
  "RUNNING",
  "PowerState/starting",
  "PowerState/running",
  "ACTIVE",
  "running",
];
export const HOST_REBOOT_STATUS = ["REBOOT", "HARD_REBOOT", "REBOOTING"];

export default defineComponent({
  props: {
    selections: {
      type: Array as PropType<Array<{ status: string }>>,
    },
  },
  setup(props) {
    const operationType = ref<Operations>(Operations.None);
    const dialogRef = ref(null);
    const isConfirmDisabled = ref(true);
    const targetHost = ref([]);
    const isLoading = ref(false);

    const previousOperationType = usePreviousState(operationType);
    const route = useRoute();
    const resourceStore = useResourceStore();

    const isDialogShow = computed(() => {
      return operationType.value !== Operations.None;
    });

    const computedTitle = computed(() => {
      if (operationType.value === Operations.None)
        return `批量${OperationsMap[previousOperationType.value]}`;
      return `批量${OperationsMap[operationType.value]}`;
    });

    const computedPreviousOperationType = computed(() => {
      switch (operationType.value) {
        case Operations.None:
          return OperationsMap[previousOperationType.value];
        case Operations.Reboot:
          return OperationsMap[Operations.Close];
        case Operations.Recycle:
          return OperationsMap[Operations.Reboot];
      }
      return OperationsMap[operationType.value];
    });

    watch(
      () => operationType.value,
      () => {
        if (operationType.value === Operations.None) return;

        const canRunHosts = [];
        const canRebootHosts = [];
        const canShutDownHosts = [];

        for (const host of props.selections) {
          const status = host.status;
          if (!HOST_RUNNING_STATUS.includes(status)) canRunHosts.push(host);
          if (!HOST_REBOOT_STATUS.includes(status)) canRebootHosts.push(host);
          if (!HOST_SHUTDOWN_STATUS.includes(status)) canShutDownHosts.push(host);
        }

        switch (operationType.value) {
          case Operations.Open: {
            targetHost.value = canRunHosts;
            break;
          }
          case Operations.Close: {
            targetHost.value = canShutDownHosts;
            break;
          }
          case Operations.Reboot: {
            targetHost.value = canShutDownHosts;
            break;
          }
          case Operations.Recycle: {
            targetHost.value = canRebootHosts;
          }
        }
        isConfirmDisabled.value = targetHost.value.length === 0;
      }
    );

    const computedContent = computed(() => {
      const allHostsNum = props.selections.length;
      const targetHostsNum = targetHost.value.length;
      if (targetHostsNum === 0) {
        return (
          <p>
            您已选择了 {allHostsNum} 台主机进行
            {`${OperationsMap[operationType.value]}`}操作, 其中
            <span class={"host_operations_blue_txt"}> {allHostsNum} </span>
            台是已{computedPreviousOperationType.value}的，不支持对其操作。
            <br />
            <span class={"host_operations_red_txt"}>
              由于所选主机均处于{computedPreviousOperationType.value}
              状态,无法进行操作。
            </span>
          </p>
        );
      } else if (targetHostsNum === allHostsNum) {
        return (
          <p>
            您已选择了 {allHostsNum} 台主机进行
            {`${OperationsMap[operationType.value]}`}操作,本次操作将对
            <span class={"host_operations_blue_txt"}> {allHostsNum} </span>
            台处于未{computedPreviousOperationType.value}
            状态的进行{`${OperationsMap[operationType.value]}`}操作。
            <br />
            <span class={"host_operations_red_txt"}>
              请确认您所选择的目标是正确的，该操作属于将对主机进行
              {`${OperationsMap[operationType.value]}`}操作。
            </span>
          </p>
        );
      } else if (allHostsNum > targetHostsNum) {
        return (
          <p>
            您已选择了 {allHostsNum} 台主机进行
            {`${OperationsMap[operationType.value]}`}操作，其中
            <span class={"host_operations_blue_txt"}> {allHostsNum - targetHostsNum} </span>
            台是已{computedPreviousOperationType.value}
            的，不支持对其操作。本次操作，将对
            <span class={"host_operations_blue_txt"}>
              {" "}
              {targetHostsNum}{" "}
            </span>
            台处于未{computedPreviousOperationType.value}状态的进行
            {`${OperationsMap[operationType.value]}`}操作。
            <br />
            <span class={"host_operations_red_txt"}>
              请确认您所选择的目标是正确的,该操作属于将对主机进行
              {`${OperationsMap[operationType.value]}`}操作
            </span>
          </p>
        );
      }
    });

    const handleConfirm = async () => {
      try {
        isLoading.value = true;
        Message({
          message: `${computedTitle.value}中, 请不要操作`,
          theme: 'warning',
          
        });
        if (operationType.value === Operations.Recycle) {
          const hostIds = targetHost.value.map(v => ({id: v.id})) as Array<Record<string, string>>;
          await resourceStore.recycledCvmsData({ infos: hostIds });
        } else {
          const hostIds = targetHost.value.map(v => v.id);
          await resourceStore.cvmOperate(operationType.value, { ids: hostIds });
        }
        Message({
          message: '操作成功',
          theme: 'success',
        });
      }
      catch(err) {
        Message({
          message: '操作失败',
          theme: 'error',
        });
      }
      finally {
        isLoading.value = false;
        operationType.value = Operations.None;
      }
    };

    const isUnderBusiness = computed(() => /^\/business\//.test(route.path));
    const operationsDisabled = computed(() => !props.selections.length);

    return () => (
      <>
        {!isUnderBusiness.value ? (
          <>
            <div>
              {Object.entries(OperationsMap).map(([opType, opName]) => (
                <Button
                  theme={opType === Operations.Open ? "primary" : undefined}
                  class="host_operations_w100 ml10"
                  onClick={() => (operationType.value = opType as Operations)}
                  disabled={operationsDisabled.value}
                >
                  {opName}
                </Button>
              ))}
            </div>

            <Dialog
              isShow={isDialogShow.value}
              quick-close={!isLoading.value}
              onClosed={() => (operationType.value = Operations.None)}
              onConfirm={handleConfirm}
              title={computedTitle.value}
              ref={dialogRef}
              width={520}
              closeIcon={!isLoading.value}
            >
              {{
                default: <p>{computedContent.value}</p>,
                footer: (
                  <>
                    <Button
                      onClick={dialogRef?.value?.handleConfirm}
                      theme="primary"
                      disabled={isConfirmDisabled.value}
                      loading={isLoading.value}
                    >
                      确定
                    </Button>
                    <Button
                      onClick={dialogRef?.value?.handleClose}
                      class="ml10"
                      disabled={isLoading.value}
                    >
                      取消
                    </Button>
                  </>
                ),
              }}
            </Dialog>
          </>
        ) : null}
      </>
    );
  },
});
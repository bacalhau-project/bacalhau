(module
  (type $t0 (func (param i32) (result i32)))
  (type $t1 (func (result i32)))
  (func $easter (export "easter") (type $t0) (param $p0 i32) (result i32)
    (local $l1 i32) (local $l2 i32) (local $l3 i32)
    (i32.and
      (i32.add
        (i32.add
          (local.tee $p0
            (i32.add
              (i32.add
                (local.tee $l1
                  (i32.rem_s
                    (i32.sub
                      (i32.add
                        (i32.shl
                          (i32.add
                            (i32.rem_s
                              (local.tee $l1
                                (i32.div_s
                                  (local.get $p0)
                                  (i32.const 100)))
                              (i32.const 4))
                            (i32.shr_s
                              (i32.shl
                                (local.tee $l3
                                  (i32.div_s
                                    (i32.shr_s
                                      (i32.shl
                                        (local.tee $l2
                                          (i32.sub
                                            (local.get $p0)
                                            (i32.mul
                                              (local.get $l1)
                                              (i32.const 100))))
                                        (i32.const 24))
                                      (i32.const 24))
                                    (i32.const 4)))
                                (i32.const 24))
                              (i32.const 24)))
                          (i32.const 1))
                        (i32.and
                          (i32.add
                            (i32.sub
                              (i32.shl
                                (local.get $l3)
                                (i32.const 2))
                              (local.get $l2))
                            (i32.const 32))
                          (i32.const 255)))
                      (local.tee $p0
                        (i32.rem_s
                          (i32.add
                            (i32.add
                              (i32.add
                                (i32.add
                                  (local.get $l1)
                                  (i32.div_s
                                    (local.get $p0)
                                    (i32.const -400)))
                                (i32.mul
                                  (local.tee $l2
                                    (i32.rem_s
                                      (local.get $p0)
                                      (i32.const 19)))
                                  (i32.const 19)))
                              (i32.div_s
                                (i32.add
                                  (i32.shl
                                    (local.get $l1)
                                    (i32.const 3))
                                  (i32.const 13))
                                (i32.const -25)))
                            (i32.const 15))
                          (i32.const 30))))
                    (i32.const 7)))
                (local.get $p0))
              (i32.and
                (i32.mul
                  (i32.div_s
                    (i32.shr_s
                      (i32.shl
                        (i32.add
                          (i32.add
                            (i32.mul
                              (local.get $p0)
                              (i32.const 11))
                            (local.get $l2))
                          (i32.mul
                            (local.get $l1)
                            (i32.const 19)))
                        (i32.const 16))
                      (i32.const 16))
                    (i32.const 433))
                  (i32.const -7))
                (i32.const 65535))))
          (i32.mul
            (i32.div_u
              (i32.and
                (i32.add
                  (local.get $p0)
                  (i32.const 90))
                (i32.const 255))
              (i32.const 25))
            (i32.const 33)))
        (i32.const 19))
      (i32.const 31)))
  (func $easter2022 (export "easter2022") (type $t1) (result i32)
    (call $easter (i32.const 2022)))
  (table $T0 1 1 funcref)
  (memory $memory (export "memory") 16)
  (global $__stack_pointer (mut i32) (i32.const 1048576))
  (global $__data_end (export "__data_end") i32 (i32.const 1048576))
  (global $__heap_base (export "__heap_base") i32 (i32.const 1048576)))

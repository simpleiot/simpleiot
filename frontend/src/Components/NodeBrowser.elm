module Components.NodeBrowser exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style


view : NodeOptions msg -> Element msg
view o =
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.globe
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            150

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        numberInput =
                            NodeInputs.nodeNumberInput opts "0"

                        boolInput =
                            NodeInputs.nodeCheckboxInput opts "0"
                    in
                    [ textInput Point.typeURL "URL" ""
                    , boolInput Point.typeDisabled "Disabled"
                    , numberInput Point.typeRotate "Rotate"
                    , boolInput Point.typeKeyboardScale "Keyboard Scale"
                    , boolInput Point.typeFullscreen "Fullscreen"
                    , boolInput Point.typeDefaultDialogs "Default Dialogs"
                    , textInput Point.typeDialogColor "Dialog Color" ""
                    , boolInput Point.typeTouchQuirk "Touch Quirk"
                    , numberInput Point.typeRetryInterval "Retry Interval"
                    , textInput Point.typeExceptionURL "Exception URL" ""
                    , boolInput Point.typeIgnoreCertErr "Ignore Cert Error"
                    , boolInput Point.typeDisableSandbox "Disable Sandbox"
                    , textInput Point.typeDebugPort "Debug Port" ""
                    , textInput Point.typeScreenResolution "Screen Resolution" ""
                    , textInput Point.typeDisplayCard "Display Card" ""
                    ]

                else
                    []
               )

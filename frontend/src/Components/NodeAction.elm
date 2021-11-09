module Components.NodeAction exposing (view)

import Api.Node as Node
import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    let
        labelWidth =
            150

        icon =
            if o.node.typ == Node.typeAction then
                Icon.trendingUp

            else
                Icon.trendingDown

        opts =
            oToInputO o labelWidth

        textInput =
            NodeInputs.nodeTextInput opts ""

        optionInput =
            NodeInputs.nodeOptionInput opts ""

        numberInput =
            NodeInputs.nodeNumberInput opts ""

        onOffInput =
            NodeInputs.nodeOnOffInput opts ""

        actionType =
            Point.getText o.node.points Point.typeActionType ""

        actionSetValue =
            actionType == Point.valueActionSetValue

        actionPlayAudio =
            actionType == Point.valueActionPlayAudio

        valueType =
            Point.getText o.node.points Point.typeValueType ""
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ icon
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeDescription "Description" ""
                    , optionInput Point.typeActionType
                        "Action"
                        [ ( Point.valueActionNotify, "notify" )
                        , ( Point.valueActionSetValue, "set node value" )
                        , ( Point.valueActionPlayAudio, "play audio" )
                        ]
                    , viewIf actionSetValue <|
                        optionInput Point.typePointType
                            "Point Type"
                            [ ( Point.typeValue, "value" )
                            , ( Point.typeValueSet, "set value (use for remote devices)" )
                            ]
                    , viewIf actionSetValue <| textInput Point.typeID "Node ID" ""
                    , viewIf actionSetValue <|
                        optionInput Point.typeValueType
                            "Point Value Type"
                            [ ( Point.valueNumber, "number" )
                            , ( Point.valueOnOff, "on/off" )
                            , ( Point.valueText, "text" )
                            ]
                    , viewIf actionSetValue <|
                        case valueType of
                            "number" ->
                                numberInput Point.typeValue "Value"

                            "onOff" ->
                                onOffInput Point.typeValue Point.typeValue "Value"

                            "text" ->
                                textInput Point.typeValue "Value" ""

                            _ ->
                                Element.none
                    , viewIf actionPlayAudio <|
                        textInput Point.typeDevice "Device" ""
                    , viewIf actionPlayAudio <|
                        numberInput Point.typeChannel "Channel"
                    , viewIf actionPlayAudio <|
                        textInput Point.typeFilePath "Wav file path" "/absolute/path/to/sound.wav"
                    ]

                else
                    []
               )
